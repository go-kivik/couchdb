package couchdb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
)

// Changes returns the changes stream for the database.
func (d *db) Changes(ctx context.Context, opts map[string]interface{}) (driver.Changes, error) {
	overrideOpts := map[string]interface{}{
		"feed":      "continuous",
		"since":     "now",
		"heartbeat": 6000,
	}
	query, err := optionsToParams(opts, overrideOpts)
	if err != nil {
		return nil, err
	}
	options := &chttp.Options{
		Query: query,
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodGet, d.path("_changes"), options)
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return newChangesRows(resp.Body), nil
}

type changesRows struct {
	body io.ReadCloser
	dec  *json.Decoder
}

func newChangesRows(r io.ReadCloser) *changesRows {
	return &changesRows{
		body: r,
	}
}

var _ driver.Changes = &changesRows{}

func (r *changesRows) Close() error {
	return r.body.Close()
}

func (r *changesRows) Next(row *driver.Change) error {
	if r.dec == nil {
		r.dec = json.NewDecoder(r.body)
	}
	if !r.dec.More() {
		_, err := r.dec.Token()
		if err != io.EOF {
			err = &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
		}
		return err
	}

	if err := r.dec.Decode(row); err != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}
	return nil
}

func (r *changesRows) LastSeq() string {
	return ""
}

func (r *changesRows) Pending() int64 {
	return 0
}

type changesNormal struct {
	rows
}

func newChangesNormal(in io.ReadCloser) *changesNormal {
	return &changesNormal{
		rows: rows{
			body:        in,
			expectedKey: "results",
		},
	}
}

func (r *changesNormal) Next(row *driver.Change) error {
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

	if err := r.nextRow(row); err != nil {
		r.closed = true
		if err == io.EOF {
			return r.finish()
		}
	}
	return nil
}

func (r *changesNormal) nextRow(row *driver.Change) error {
	if !r.dec.More() {
		if err := consumeDelim(r.dec, json.Delim(']')); err != nil {
			return err
		}
		return io.EOF
	}
	if err := r.dec.Decode(row); err != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}
	return nil
}
