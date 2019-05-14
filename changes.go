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
	query, err := optionsToParams(opts)
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

type continuousChangesParser struct{}

func (p *continuousChangesParser) decodeItem(i interface{}, dec *json.Decoder) error {
	row := i.(*driver.Change)
	ch := &change{Change: row}
	if err := dec.Decode(ch); err != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}
	ch.Change.Seq = string(ch.Seq)
	return nil
}

type changesRows struct {
	*iter
}

func newChangesRows(r io.ReadCloser) *changesRows {
	return &changesRows{
		iter: newIter(nil, "", r, &continuousChangesParser{}),
	}
}

var _ driver.Changes = &changesRows{}

type change struct {
	*driver.Change
	Seq sequenceID `json:"seq"`
}

func (r *changesRows) Next(row *driver.Change) error {
	return r.iter.next(row)
}

// LastSeq returns an empty string.
func (r *changesRows) LastSeq() string {
	return ""
}

// Pending returns 0.
func (r *changesRows) Pending() int64 {
	return 0
}
