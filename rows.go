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
	*iter
	offset    int64
	totalRows int64
	updateSeq string
	warning   string
	bookmark  string
}

var _ driver.Rows = &rows{}

func newRows(in io.ReadCloser) *rows {
	r := &rows{
		iter: newIter("rows", in),
	}
	r.decodeRow = func(row interface{}) error {
		return r.dec.Decode(row)
	}
	r.iter.parseMeta = func(_ *json.Decoder, key string) error {
		return r.parseMeta(key)
	}
	return r
}

func newFindRows(in io.ReadCloser) *rows {
	r := &rows{
		iter: newIter("docs", in),
	}
	r.decodeRow = func(i interface{}) error {
		row := i.(*driver.Row)
		return r.dec.Decode(&row.Doc)
	}
	r.iter.parseMeta = func(_ *json.Decoder, key string) error {
		return r.parseMeta(key)
	}
	return r
}

func newBulkGetRows(in io.ReadCloser) *rows {
	r := &rows{
		iter: newIter("results", in),
	}
	r.decodeRow = func(i interface{}) error {
		row := i.(*driver.Row)
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
	r.iter.parseMeta = func(_ *json.Decoder, key string) error {
		return r.parseMeta(key)
	}
	return r
}

func (r *rows) Offset() int64 {
	return r.offset
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

func (r *rows) UpdateSeq() string {
	return r.updateSeq
}

func (r *rows) Next(row *driver.Row) error {
	return r.iter.next(row)
}

// parseMeta parses result metadata
func (r *rows) parseMeta(key string) error {
	switch key {
	case "update_seq":
		return r.readUpdateSeq()
	case "offset":
		return r.dec.Decode(&r.offset)
	case "total_rows":
		return r.dec.Decode(&r.totalRows)
	case "warning":
		return r.dec.Decode(&r.warning)
	case "bookmark":
		return r.dec.Decode(&r.bookmark)
	}
	return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("Unexpected key: %s", key)}
}

func (r *rows) readUpdateSeq() error {
	var raw json.RawMessage
	if err := r.dec.Decode(&raw); err != nil {
		return err
	}
	r.updateSeq = string(bytes.Trim(raw, `""`))
	return nil
}
