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

type change struct {
	*driver.Change
	Seq sequenceID `json:"seq"`
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

	ch := &change{Change: row}
	if err := r.dec.Decode(ch); err != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}
	ch.Change.Seq = string(ch.Seq)
	return nil
}

// LastSeq returns an empty string.
func (r *changesRows) LastSeq() string {
	return ""
}

// Pending returns 0.
func (r *changesRows) Pending() int64 {
	return 0
}
