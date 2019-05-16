package couchdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/go-kivik/kivik"
)

type parser interface {
	decodeItem(interface{}, *json.Decoder) error
	parseMeta(interface{}, *json.Decoder, string) error
}

type iter struct {
	meta        interface{}
	expectedKey string
	body        io.ReadCloser
	parser      parser

	dec    *json.Decoder
	mu     sync.RWMutex
	closed bool
}

func newIter(meta interface{}, expectedKey string, body io.ReadCloser, parser parser) *iter {
	return &iter{
		meta:        meta,
		expectedKey: expectedKey,
		body:        body,
		parser:      parser,
	}
}

func (i *iter) next(row interface{}) error {
	i.mu.RLock()
	if i.closed {
		i.mu.RUnlock()
		return io.EOF
	}
	i.mu.RUnlock()
	if i.dec == nil {
		// We havenn't begun yet
		i.dec = json.NewDecoder(i.body)
		if err := i.begin(); err != nil {
			return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
		}
	}

	err := i.nextRow(row)
	if err != nil {
		_ = i.Close()
		if err == io.EOF {
			return i.finish()
		}
	}
	return err
}

// begin parses the top-level of the result object; until rows
func (i *iter) begin() error {
	if i.expectedKey == "" {
		return nil
	}
	// consume the first '{'
	if err := consumeDelim(i.dec, json.Delim('{')); err != nil {
		return err
	}
	for {
		t, err := i.dec.Token()
		if err != nil {
			// I can't find a test case to trigger this, so it remains uncovered.
			return err
		}
		key, ok := t.(string)
		if !ok {
			// The JSON parser should never permit this
			return fmt.Errorf("Unexpected token: (%T) %v", t, t)
		}
		if key == i.expectedKey {
			// Consume the first '['
			return consumeDelim(i.dec, json.Delim('['))
		}
		if err := i.parser.parseMeta(i.meta, i.dec, key); err != nil {
			return err
		}
	}
}

func (i *iter) finish() error {
	if i.expectedKey == "" {
		_, err := i.dec.Token()
		if err != io.EOF {
			return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
		}
		return nil
	}
	if err := consumeDelim(i.dec, json.Delim(']')); err != nil {
		return err
	}
	for {
		t, err := i.dec.Token()
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
			if err := i.parser.parseMeta(i.meta, i.dec, v); err != nil {
				return err
			}
		default:
			// This should never happen, as the JSON parser would never get
			// this far.
			return fmt.Errorf("Unexpected JSON token: (%T) '%s'", t, t)
		}
	}
}

func (i *iter) nextRow(row interface{}) error {
	if !i.dec.More() {
		return io.EOF
	}
	return i.parser.decodeItem(row, i.dec)
}

func (i *iter) Close() error {
	i.mu.Lock()
	i.closed = true
	i.mu.Unlock()
	return i.body.Close()
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
