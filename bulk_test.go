package couchdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"gitlab.com/flimzy/testy"

	kivik "github.com/go-kivik/kivik/v3"
	"github.com/go-kivik/kivik/v3/driver"
)

func TestBulkDocs(t *testing.T) {
	tests := []struct {
		name    string
		db      *db
		docs    []interface{}
		options map[string]interface{}
		status  int
		err     string
	}{
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: http.StatusBadGateway,
			err:    "Post http://example.com/testdb/_bulk_docs: net error",
		},
		{
			name: "JSON encoding error",
			db: newTestDB(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			docs:   []interface{}{make(chan int)},
			status: http.StatusBadRequest,
			err:    "Post http://example.com/testdb/_bulk_docs: json: unsupported type: chan int",
		},
		{
			name: "docs rejected",
			db: newTestDB(&http.Response{
				StatusCode: http.StatusExpectationFailed,
				Body:       ioutil.NopCloser(strings.NewReader("[]")),
			}, nil),
			docs:   []interface{}{1, 2, 3},
			status: http.StatusExpectationFailed,
			err:    "Expectation Failed: one or more document was rejected",
		},
		{
			name: "error response",
			db: newTestDB(&http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			docs:   []interface{}{1, 2, 3},
			status: http.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "invalid JSON response",
			db: newTestDB(&http.Response{
				StatusCode: http.StatusCreated,
				Body:       ioutil.NopCloser(strings.NewReader("invalid json")),
			}, nil),
			docs:   []interface{}{1, 2, 3},
			status: http.StatusBadGateway,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "unexpected response code",
			db: newTestDB(&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("[]")),
			}, nil),
			docs: []interface{}{1, 2, 3},
		},
		{
			name:    "new_edits",
			options: map[string]interface{}{"new_edits": true},
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close() // nolint: errcheck
				var body struct {
					NewEdits bool `json:"new_edits"`
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					return nil, err
				}
				if !body.NewEdits {
					return nil, errors.New("`new_edits` not set")
				}
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       ioutil.NopCloser(strings.NewReader("[]")),
				}, nil
			}),
		},
		{
			name:    "full commit",
			options: map[string]interface{}{OptionFullCommit: true},
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close() // nolint: errcheck
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					return nil, err
				}
				if _, ok := body[OptionFullCommit]; ok {
					return nil, errors.New("Full Commit key found in body")
				}
				if value := req.Header.Get("X-Couch-Full-Commit"); value != "true" {
					return nil, errors.New("X-Couch-Full-Commit not set to true")
				}
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       ioutil.NopCloser(strings.NewReader("[]")),
				}, nil
			}),
		},
		{
			name:    "invalid full commit type",
			db:      &db{},
			options: map[string]interface{}{OptionFullCommit: 123},
			status:  http.StatusBadRequest,
			err:     "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.db.BulkDocs(context.Background(), test.docs, test.options)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestBulkNext(t *testing.T) {
	tests := []struct {
		name     string
		results  *bulkResults
		status   int
		err      string
		expected *driver.BulkResult
	}{
		{
			name: "no results",
			results: func() *bulkResults {
				r, err := newBulkResults(Body(`[]`))
				if err != nil {
					t.Fatal(err)
				}
				return r
			}(),
			status: http.StatusInternalServerError,
			err:    "EOF",
		},
		{
			name: "closing delimiter missing",
			results: func() *bulkResults {
				r, err := newBulkResults(Body(`[`))
				if err != nil {
					t.Fatal(err)
				}
				return r
			}(),
			status: http.StatusBadGateway,
			err:    "EOF",
		},
		{
			name: "invalid doc json",
			results: func() *bulkResults {
				r, err := newBulkResults(Body(`[{foo}]`))
				if err != nil {
					t.Fatal(err)
				}
				return r
			}(),
			status: http.StatusBadGateway,
			err:    "invalid character 'f' looking for beginning of object key string",
		},
		{
			name: "successful update",
			results: func() *bulkResults {
				r, err := newBulkResults(Body(`[{"id":"foo","rev":"1-xxx"}]`))
				if err != nil {
					t.Fatal(err)
				}
				return r
			}(),
			expected: &driver.BulkResult{
				ID:  "foo",
				Rev: "1-xxx",
			},
		},
		{
			name: "conflict",
			results: func() *bulkResults {
				r, err := newBulkResults(Body(`[{"id":"foo","error":"conflict","reason":"annoying conflict"}]`))
				if err != nil {
					t.Fatal(err)
				}
				return r
			}(),
			expected: &driver.BulkResult{
				ID:    "foo",
				Error: &kivik.Error{HTTPStatus: http.StatusConflict, FromServer: true, Err: errors.New("annoying conflict")},
			},
		},
		{
			name: "unknown error",
			results: func() *bulkResults {
				r, err := newBulkResults(Body(`[{"id":"foo","error":"foo","reason":"foo is erroneous"}]`))
				if err != nil {
					t.Fatal(err)
				}
				return r
			}(),
			expected: &driver.BulkResult{
				ID:    "foo",
				Error: &kivik.Error{HTTPStatus: http.StatusInternalServerError, FromServer: true, Err: errors.New("foo is erroneous")},
			},
		},
		{
			name: "read error",
			results: func() *bulkResults {
				r, err := newBulkResults(ioutil.NopCloser(testy.ErrorReader("[", errors.New("read error"))))
				if err != nil {
					t.Fatal(err)
				}
				return r
			}(),
			status: http.StatusBadGateway,
			err:    "read error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(driver.BulkResult)
			err := test.results.Next(result)
			testy.StatusError(t, test.err, test.status, err)
			if d := testy.DiffInterface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

type closeTracker struct {
	closed bool
	io.ReadCloser
}

func (c *closeTracker) Close() error {
	c.closed = true
	return c.ReadCloser.Close()
}

func TestBulkClose(t *testing.T) {
	body := &closeTracker{
		ReadCloser: Body(`[{"id":"foo","error":"foo","reason":"foo is erroneous"}]`),
	}
	r, err := newBulkResults(body)
	if err != nil {
		t.Fatal(err)
	}
	if e := r.Close(); e != nil {
		t.Fatal(e)
	}
	if !body.closed {
		t.Errorf("Failed to close")
	}
}
