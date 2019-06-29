package couchdb

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
)

func TestChanges(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]interface{}
		db      *db
		status  int
		err     string
		etag    string
	}{
		{
			name:    "invalid options",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadAPICall,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name:    "eventsource",
			options: map[string]interface{}{"feed": "eventsource"},
			status:  kivik.StatusBadRequest,
			err:     "kivik: eventsource feed not supported, use 'continuous'",
		},
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb/_changes: net error",
		},
		{
			name:    "continuous",
			db:      newTestDB(nil, errors.New("net error")),
			options: map[string]interface{}{"feed": "continuous"},
			status:  kivik.StatusNetworkError,
			err:     "Get http://example.com/testdb/_changes?feed=continuous: net error",
		},
		{
			name: "error response",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       Body(""),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "success 1.6.1",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Transfer-Encoding": {"chunked"},
					"Server":            {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":              {"Fri, 27 Oct 2017 14:43:57 GMT"},
					"Content-Type":      {"text/plain; charset=utf-8"},
					"Cache-Control":     {"must-revalidate"},
					"ETag":              {`"etag-foo"`},
				},
				Body: Body(`{"seq":3,"id":"43734cf3ce6d5a37050c050bb600006b","changes":[{"rev":"2-185ccf92154a9f24a4f4fd12233bf463"}],"deleted":true}
                    `),
			}, nil),
			etag: "etag-foo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ch, err := test.db.Changes(context.Background(), test.options)
			if ch != nil {
				defer ch.Close()
			}
			testy.StatusError(t, test.err, test.status, err)
			if etag := ch.ETag(); etag != test.etag {
				t.Errorf("Unexpected ETag: %s", etag)
			}
		})
	}
}

func TestChangesNext(t *testing.T) {
	tests := []struct {
		name     string
		changes  *changesRows
		status   int
		err      string
		expected *driver.Change
	}{
		{
			name:    "invalid json",
			changes: newChangesRows(context.TODO(), "", Body("invalid json"), ""),
			status:  kivik.StatusBadResponse,
			err:     "invalid character 'i' looking for beginning of value",
		},
		{
			name: "success",
			changes: newChangesRows(context.TODO(), "", Body(`{"seq":3,"id":"43734cf3ce6d5a37050c050bb600006b","changes":[{"rev":"2-185ccf92154a9f24a4f4fd12233bf463"}],"deleted":true}
                `), ""),
			expected: &driver.Change{
				ID:      "43734cf3ce6d5a37050c050bb600006b",
				Seq:     "3",
				Deleted: true,
				Changes: []string{"2-185ccf92154a9f24a4f4fd12233bf463"},
			},
		},
		{
			name:    "read error",
			changes: newChangesRows(context.TODO(), "", ioutil.NopCloser(testy.ErrorReader("", errors.New("read error"))), ""),
			status:  http.StatusBadGateway,
			err:     "read error",
		},
		{
			name:     "end of input",
			changes:  newChangesRows(context.TODO(), "", Body(``), ""),
			expected: &driver.Change{},
			status:   http.StatusInternalServerError,
			err:      "EOF",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := new(driver.Change)
			err := test.changes.Next(row)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, row); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestChangesClose(t *testing.T) {
	body := &closeTracker{ReadCloser: Body("foo")}
	feed := newChangesRows(context.TODO(), "", body, "")
	_ = feed.Close()
	if !body.closed {
		t.Errorf("Failed to close")
	}
}
