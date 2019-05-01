package couchdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
)

type rows struct {
	offset      int64
	totalRows   int64
	pending     int64
	updateSeq   string
	lastSeq     string
	warning     string
	bookmark    string
	body        io.ReadCloser
	dec         *json.Decoder
	expectedKey string
	decodeRow   func(*driver.Row) error
	// closed is true after all rows have been processed
	closed bool
}

var _ driver.Rows = &rows{}

func newRows(in io.ReadCloser) *rows {
	r := &rows{
		body:        in,
		expectedKey: "rows",
	}
	r.decodeRow = func(row *driver.Row) error {
		return r.dec.Decode(row)
	}
	return r
}

func newFindRows(in io.ReadCloser) *rows {
	r := &rows{
		body:        in,
		expectedKey: "docs",
	}
	r.decodeRow = func(row *driver.Row) error {
		return r.dec.Decode(&row.Doc)
	}
	return r
}

func newBulkGetRows(in io.ReadCloser) *rows {
	r := &rows{
		body:        in,
		expectedKey: "results",
	}
	r.decodeRow = func(row *driver.Row) error {
		var result bulkResult
		if err := r.dec.Decode(&result); err != nil {
			return err
		}
		row.ID = result.ID
		row.Doc = result.Docs[0].Doc
		row.Error = nil
		if err := result.Docs[0].Error; err != nil {
			row.Error = err
		}
		return nil
	}
	return r
}

func (r *rows) Offset() int64 {
	return r.offset
}

func (r *rows) Pending() int64 {
	return r.pending
}

func (r *rows) TotalRows() int64 {
	return r.totalRows
}

func (r *rows) Warning() string {
	return r.warning
}

func (r *rows) Bookmark() string {
	return r.bookmark
}

func (r *rows) LastSeq() driver.SequenceID {
	return driver.SequenceID(r.lastSeq)
}

func (r *rows) UpdateSeq() string {
	return r.updateSeq
}

func (r *rows) Close() error {
	return r.body.Close()
}

func (r *rows) Next(row *driver.Row) error {
	if r.closed {
		return io.EOF
	}
	if r.dec == nil {
		// We haven't begun yet
		r.dec = json.NewDecoder(r.body)
		// consume the first '{'
		if err := consumeDelim(r.dec, json.Delim('{')); err != nil {
			return err
		}
		if err := r.begin(); err != nil {
			return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
		}
	}

	err := r.nextRow(row)
	if err != nil {
		r.closed = true
		if err == io.EOF {
			return r.finish()
		}
	}
	return err
}

// begin parses the top-level of the result object; until rows
func (r *rows) begin() error {
	for {
		t, err := r.dec.Token()
		if err != nil {
			// I can't find a test case to trigger this, so it remains uncovered.
			return err
		}
		key, ok := t.(string)
		if !ok {
			// The JSON parser should never permit this
			return fmt.Errorf("Unexpected token: (%T) %v", t, t)
		}
		if key == r.expectedKey {
			// Consume the first '['
			return consumeDelim(r.dec, json.Delim('['))
		}
		if err := r.parseMeta(key); err != nil {
			return err
		}
	}
}

func (r *rows) finish() error {
	for {
		t, err := r.dec.Token()
		if err != nil {
			return err
		}
		switch v := t.(type) {
		case json.Delim:
			if v != json.Delim('}') {
				// This should never happen, as the JSON parser should prevent it.
				return fmt.Errorf("Unexpected JSON delimiter: %c", v)
			}
		case string:
			if err := r.parseMeta(v); err != nil {
				return err
			}
		default:
			// This should never happen, as the JSON parser would never get
			// this far.
			return fmt.Errorf("Unexpected JSON token: (%T) '%s'", t, t)
		}
	}
}

// parseMeta parses result metadata
func (r *rows) parseMeta(key string) error {
	switch key {
	case "update_seq":
		return r.readSeq(&r.updateSeq)
	case "last_seq":
		return r.readSeq(&r.lastSeq)
	case "offset":
		return r.dec.Decode(&r.offset)
	case "pending":
		return r.dec.Decode(&r.pending)
	case "total_rows":
		return r.dec.Decode(&r.totalRows)
	case "warning":
		return r.dec.Decode(&r.warning)
	case "bookmark":
		return r.dec.Decode(&r.bookmark)
	}
	return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("Unexpected key: %s", key)}
}

func (r *rows) readSeq(seq *string) error {
	var raw json.RawMessage
	if err := r.dec.Decode(&raw); err != nil {
		return err
	}
	*seq = string(bytes.Trim(raw, `""`))
	return nil
}

func (r *rows) nextRow(row *driver.Row) error {
	if !r.dec.More() {
		if err := consumeDelim(r.dec, json.Delim(']')); err != nil {
			return err
		}
		return io.EOF
	}
	return r.decodeRow(row)
}

// consumeDelim consumes the expected delimiter from the stream, or returns an
// error if an unexpected token was found.
func consumeDelim(dec *json.Decoder, expectedDelim json.Delim) error {
	t, err := dec.Token()
	if err != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}
	d, ok := t.(json.Delim)
	if !ok {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("Unexpected token %T: %v", t, t)}
	}
	if d != expectedDelim {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("Unexpected JSON delimiter: %c", d)}
	}
	return nil
}
