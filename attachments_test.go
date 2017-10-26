package couchdb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/testy"
)

func TestPutAttachment(t *testing.T) {
	tests := []struct {
		name                     string
		db                       *db
		id, rev, filename, ctype string
		body                     io.Reader
		newRev                   string
		status                   int
		err                      string
	}{
		{
			name:   "missing docID",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name: "missing filename",
			id:   "foo", rev: "1-xxx",
			status: kivik.StatusBadRequest,
			err:    "kivik: filename required",
		},
		{
			name: "missing content type",
			id:   "foo", rev: "1-xxx", filename: "x.jpg",
			status: kivik.StatusBadRequest,
			err:    "kivik: contentType required",
		},
		{
			name: "network error",
			id:   "foo", rev: "1-xxx", filename: "x.jpg", ctype: "image/jpeg",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusInternalServerError,
			err:    "Put http://example.com/testdb/foo/x.jpg?rev=1-xxx: net error",
		},
		{
			name:     "1.6.1",
			id:       "foo",
			rev:      "1-4c6114c65e295552ab1019e2b046b10e",
			filename: "foo.txt",
			ctype:    "text/plain",
			body:     strings.NewReader("Hello, World!"),
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close() // nolint: errcheck
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "text/plain" {
					return nil, fmt.Errorf("Unexpected Content-Type: %s", ct)
				}
				expectedRev := "1-4c6114c65e295552ab1019e2b046b10e"
				if rev := req.URL.Query().Get("rev"); rev != expectedRev {
					return nil, fmt.Errorf("Unexpected rev: %s", rev)
				}
				body, err := ioutil.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				expected := "Hello, World!"
				if string(body) != expected {
					t.Errorf("Unexpected body:\n%s\n", string(body))
				}
				return &http.Response{
					StatusCode: 201,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Location":       {"http://localhost:5984/foo/foo/foo.txt"},
						"ETag":           {`"2-8ee3381d24ee4ac3e9f8c1f6c7395641"`},
						"Date":           {"Thu, 26 Oct 2017 20:51:35 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"66"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"ok":true,"id":"foo","rev":"2-8ee3381d24ee4ac3e9f8c1f6c7395641"}`),
				}, nil
			}),
			newRev: "2-8ee3381d24ee4ac3e9f8c1f6c7395641",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRev, err := test.db.PutAttachment(context.Background(), test.id, test.rev, test.filename, test.ctype, test.body)
			testy.StatusError(t, test.err, test.status, err)
			if newRev != test.newRev {
				t.Errorf("Expected %s, got %s\n", test.newRev, newRev)
			}
		})
	}
}
