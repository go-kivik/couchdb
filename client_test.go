package couchdb

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/testy"
	"github.com/go-kivik/kiviktest/kt"
)

func TestAllDBs(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		expected []string
		status   int
		err      string
	}{
		{
			name:   "network error",
			client: newTestClient(nil, errors.New("net error")),
			status: 500,
			err:    "Get http://example.com/_all_dbs: net error",
		},
		{
			name: "2.0.0",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 15:15:07 GMT"},
					"Content-Type":        {"application/json"},
					"ETag":                {`"33UVNAZU752CYNGBBTMWQFP7U"`},
					"Transfer-Encoding":   {"chunked"},
					"X-Couch-Request-ID":  {"ab5cd97c3e"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: Body(`["_global_changes","_replicator","_users"]`),
			}, nil),
			expected: []string{"_global_changes", "_replicator", "_users"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.AllDBs(context.Background(), nil)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDBExists(t *testing.T) {
	tests := []struct {
		name   string
		client *client
		dbName string
		exists bool
		status int
		err    string
	}{
		{
			name:   "no db specified",
			status: kivik.StatusBadRequest,
			err:    "kivik: dbName required",
		},
		{
			name:   "network error",
			dbName: "foo",
			client: newTestClient(nil, errors.New("net error")),
			status: 500,
			err:    "Head http://example.com/foo: net error",
		},
		{
			name:   "not found, 1.6.1",
			dbName: "foox",
			client: newTestClient(&http.Response{
				StatusCode: 404,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 15:09:19 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"44"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(""),
			}, nil),
			exists: false,
		},
		{
			name:   "exists, 1.6.1",
			dbName: "foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 15:09:19 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"229"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(""),
			}, nil),
			exists: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := test.client.DBExists(context.Background(), test.dbName, nil)
			testy.StatusError(t, test.err, test.status, err)
			if exists != test.exists {
				t.Errorf("Unexpected result: %t", exists)
			}
		})
	}
}

func TestCreateAndDestroyDB(t *testing.T) {
	client := getClient(t)
	dbName := kt.TestDBName(t)
	defer client.DestroyDB(context.Background(), dbName, nil) // nolint: errcheck
	if err := client.CreateDB(context.Background(), dbName, nil); err != nil {
		t.Errorf("Create failed: %s", err)
	}
	if err := client.DestroyDB(context.Background(), dbName, nil); err != nil {
		t.Errorf("Destroy failed: %s", err)
	}
}
