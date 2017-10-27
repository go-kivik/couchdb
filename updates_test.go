package couchdb

import (
	"errors"
	"net/http"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestDBUpdates(t *testing.T) {
	tests := []struct {
		name   string
		client *client
		status int
		err    string
	}{
		{
			name:   "network error",
			client: newTestClient(nil, errors.New("net error")),
			status: 500,
			err:    "Get http://example.com/_db_updates?feed=continuous&since=now: net error",
		},
		{
			name: "error response",
			client: newTestClient(&http.Response{
				StatusCode: 400,
				Body:       Body(""),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "Success 1.6.1",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Transfer-Encoding": {"chunked"},
					"Server":            {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":              {"Fri, 27 Oct 2017 19:55:43 GMT"},
					"Content-Type":      {"application/json"},
					"Cache-Control":     {"must-revalidate"},
				},
				Body: Body(""),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.DBUpdates()
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := result.(*couchUpdates); !ok {
				t.Errorf("Unexpected type returned: %t", result)
			}
		})
	}
}

func TestUpdatesNext(t *testing.T) {
	tests := []struct {
		name     string
		updates  *couchUpdates
		status   int
		err      string
		expected *driver.DBUpdate
	}{
		{
			name:    "consumed feed",
			updates: newUpdates(Body("")),
			status:  500,
			err:     "EOF",
		},
		{
			name:    "read feed",
			updates: newUpdates(Body(`{"db_name":"mailbox","type":"created","seq":"1-g1AAAAFReJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuDOZExFyjAnmJhkWaeaIquGIf2JAUgmWQPMiGRAZcaB5CaePxqEkBq6vGqyWMBkgwNQAqobD4h"},`)),
			expected: &driver.DBUpdate{
				DBName: "mailbox",
				Type:   "created",
				Seq:    "1-g1AAAAFReJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuDOZExFyjAnmJhkWaeaIquGIf2JAUgmWQPMiGRAZcaB5CaePxqEkBq6vGqyWMBkgwNQAqobD4h",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(driver.DBUpdate)
			err := test.updates.Next(result)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestUpdatesClose(t *testing.T) {
	body := &closeTracker{ReadCloser: Body("")}
	u := newUpdates(body)
	if err := u.Close(); err != nil {
		t.Fatal(err)
	}
	if !body.closed {
		t.Errorf("Failed to close")
	}
}
