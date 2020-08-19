// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package couchdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-kivik/couchdb/v4/chttp"
	kivik "github.com/go-kivik/kivik/v4"
	"github.com/go-kivik/kivik/v4/driver"
)

type bulkResults struct {
	body io.ReadCloser
	dec  *json.Decoder
}

var _ driver.BulkResults = &bulkResults{}

func newBulkResults(body io.ReadCloser) (*bulkResults, error) {
	dec := json.NewDecoder(body)
	// Consume the opening '[' char
	if err := consumeDelim(dec, json.Delim('[')); err != nil {
		return nil, err
	}
	return &bulkResults{
		body: body,
		dec:  dec,
	}, nil
}

func (r *bulkResults) Next(update *driver.BulkResult) error {
	if !r.dec.More() {
		if err := consumeDelim(r.dec, json.Delim(']')); err != nil {
			return err
		}
		return io.EOF
	}
	var updateResult struct {
		ID     string `json:"id"`
		Rev    string `json:"rev"`
		Error  string `json:"error"`
		Reason string `json:"reason"`
	}
	if err := r.dec.Decode(&updateResult); err != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}
	update.ID = updateResult.ID
	update.Rev = updateResult.Rev
	update.Error = nil
	if updateResult.Error != "" {
		var status int
		switch updateResult.Error {
		case "conflict":
			status = http.StatusConflict
		default:
			status = http.StatusInternalServerError
		}
		update.Error = &kivik.Error{HTTPStatus: status, Err: errors.New(updateResult.Reason)}
	}
	return nil
}

func (r *bulkResults) Close() error {
	return r.body.Close()
}

func (d *db) BulkDocs(ctx context.Context, docs []interface{}, options map[string]interface{}) (driver.BulkResults, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	fullCommit, err := fullCommit(options)
	if err != nil {
		return nil, err
	}
	options["docs"] = docs
	opts := &chttp.Options{
		GetBody:    chttp.BodyEncoder(options),
		FullCommit: fullCommit,
	}
	resp, err := d.Client.DoReq(ctx, http.MethodPost, d.path("_bulk_docs"), opts)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusCreated:
		// Nothing to do
	case http.StatusExpectationFailed:
		err = &chttp.HTTPError{
			Response: resp,
			Reason:   "one or more document was rejected",
		}
	default:
		// All other errors can consume the response body and return immediately
		if e := chttp.ResponseError(resp); e != nil {
			return nil, e
		}
	}
	results, bulkErr := newBulkResults(resp.Body)
	if bulkErr != nil {
		return nil, bulkErr
	}
	return results, err
}
