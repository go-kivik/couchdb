package couchdb

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/testy"
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
			status: 500,
			err:    "Post http://example.com/testdb/_bulk_docs: net error",
		},
		{
			name: "JSON encoding error",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			docs:   []interface{}{make(chan int)},
			status: 500,
			err:    "json: unsupported type: chan int",
		},
		{
			name: "docs rejected",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusExpectationFailed,
				Body:       ioutil.NopCloser(strings.NewReader("[]")),
			}, nil),
			docs:   []interface{}{1, 2, 3},
			status: kivik.StatusExpectationFailed,
			err:    "Expectation Failed: one or more document was rejected",
		},
		{
			name: "bad request",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			docs:   []interface{}{1, 2, 3},
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "invalid JSON response",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusCreated,
				Body:       ioutil.NopCloser(strings.NewReader("invalid json")),
			}, nil),
			docs:   []interface{}{1, 2, 3},
			status: 500,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "unexpected response code",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("[]")),
			}, nil),
			docs: []interface{}{1, 2, 3},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.db.BulkDocs(context.Background(), test.docs, test.options)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}
