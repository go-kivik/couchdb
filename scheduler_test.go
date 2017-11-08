package couchdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestSRUpdate(t *testing.T) {
	tests := []struct {
		name     string
		rep      *schedulerReplication
		status   int
		err      string
		expected *driver.ReplicationInfo
	}{
		{
			name: "network error",
			rep: &schedulerReplication{
				database: "_replicator",
				docID:    "foo",
				db:       newTestDB(nil, errors.New("net error")),
			},
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_scheduler/docs/_replicator/foo: net error",
		},
		{
			name: "real example",
			rep: &schedulerReplication{
				database: "_replicator",
				docID:    "foo2",
				db: newTestDB(&http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Server":         {"CouchDB/2.1.0 (Erlang OTP/17)"},
						"Date":           {"Thu, 09 Nov 2017 15:23:20 GMT"},
						"Content-Type":   {"application/json"},
						"Content-Length": {"687"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"database":"_replicator","doc_id":"foo2","id":null,"source":"http://localhost:5984/foo/","target":"http://localhost:5984/bar/","state":"completed","error_count":0,"info":{"revisions_checked":23,"missing_revisions_found":23,"docs_read":23,"docs_written":23,"changes_pending":null,"doc_write_failures":0,"checkpointed_source_seq":"27-g1AAAAIbeJyV0EsOgjAQBuAGMOLCM-gRSoUKK7mJ9kWQYLtQ13oTvYneRG-CfZAYSUjqZppM5v_SmRYAENchB3OppOKilKpWx1Or2wEBdNF1XVOHJD7oxnTFKMOcDYdH4nSpK930wsQKAmYIVdBXKI2w_RGQyFJYFb7CzgiXXgDuDywXKUk4mJ0lF9VeCj6SlpGu4KofDdyMEFoBk3QtMt87OOXulIdRAqvABHPO0F_K0ymv7zYU5UVe-W_zdoK9R2QFxhjBUAwzzQch86VT"},"start_time":"2017-11-01T21:05:03Z","last_updated":"2017-11-01T21:05:06Z"}`),
				}, nil),
			},
			expected: &driver.ReplicationInfo{
				DocsRead:    23,
				DocsWritten: 23,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var result driver.ReplicationInfo
			err := test.rep.Update(context.Background(), &result)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, &result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestRepInfoUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *repInfo
		err      string
	}{
		{
			name:     "null",
			input:    "null",
			expected: &repInfo{},
		},
		{
			name:  "error string",
			input: `"db_not_found: could not open foo"`,
			expected: &repInfo{
				Error: &replicationError{
					status: 404,
					reason: "db_not_found: could not open foo",
				},
			},
		},
		{
			name:  "stats",
			input: `{"revisions_checked":23,"missing_revisions_found":23,"docs_read":23,"docs_written":23,"changes_pending":null,"doc_write_failures":0,"checkpointed_source_seq":"27-g1AAAAIbeJyV0EsOgjAQBuAGMOLCM-gRSoUKK7mJ9kWQYLtQ13oTvYneRG-CfZAYSUjqZppM5v_SmRYAENchB3OppOKilKpWx1Or2wEBdNF1XVOHJD7oxnTFKMOcDYdH4nSpK930wsQKAmYIVdBXKI2w_RGQyFJYFb7CzgiXXgDuDywXKUk4mJ0lF9VeCj6SlpGu4KofDdyMEFoBk3QtMt87OOXulIdRAqvABHPO0F_K0ymv7zYU5UVe-W_zdoK9R2QFxhjBUAwzzQch86VT"}`,
			expected: &repInfo{
				DocsRead:         23,
				DocsWritten:      23,
				DocWriteFailures: 0,
			},
		},
		{
			name:  "invalid stats object",
			input: `{"docs_written":"chicken"}`,
			err:   "^json: cannot unmarshal string into Go ",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(repInfo)
			err := json.Unmarshal([]byte(test.input), result)
			testy.ErrorRE(t, test.err, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestGetReplicationsFromScheduler(t *testing.T) {
	tests := []struct {
		name     string
		options  map[string]interface{}
		client   *client
		expected []*schedulerReplication
		status   int
		err      string
	}{
		{
			name: "scheduler not supported, 2.0",
			client: newTestClient(&http.Response{
				StatusCode: 404,
				Header: http.Header{
					"Cache-Control":       {"must-revalidate"},
					"Content-Length":      {"58"},
					"Content-Type":        {"application/json"},
					"Date":                {"Wed, 08 Nov 2017 17:52:38 GMT"},
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"X-Couch-Request-ID":  {"8b9574a6f8"},
					"X-CouchDB-Body-Time": {"0"},
				},
				ContentLength: 58,
				Body:          Body(`{"error":"not_found","reason":"Database does not exist."}`),
			}, nil),
			status: kivik.StatusNotImplemented,
			err:    "_scheduler interface not implemented",
		},
		{
			name: "scheduler not supported, 1.6",
			client: newTestClient(&http.Response{
				StatusCode: 400,
				Header: http.Header{
					"Cache-Control":       {"must-revalidate"},
					"Content-Length":      {"201"},
					"Content-Type":        {"application/json"},
					"Date":                {"Wed, 08 Nov 2017 17:52:38 GMT"},
					"Server":              {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"X-Couch-Request-ID":  {"8b9574a6f8"},
					"X-CouchDB-Body-Time": {"0"},
				},
				ContentLength: 58,
				Body:          Body(`{"error":"illegal_database_name","reason":"Name: '_scheduler'. Only lowercase characters (a-z), digits (0-9), and any of the characters _, $, (, ), +, -, and / are allowed. Must begin with a letter."}`),
			}, nil),
			status: kivik.StatusNotImplemented,
			err:    "_scheduler interface not implemented",
		},
		{
			name:   "network error",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_scheduler/docs: net error",
		},
		{
			name:    "invalid options",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name: "valid response, 2.1.0",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Wed, 08 Nov 2017 18:04:11 GMT"},
					"Content-Type":        {"application/json"},
					"Transfer-Encoding":   {"chunked"},
					"Cache-Control":       {"must-revalidate"},
					"X-CouchDB-Body-Time": {"0"},
					"X-Couch-Request-ID":  {"6d47891c37"},
				},
				Body: Body(`{"total_rows":2,"offset":0,"docs":[
{"database":"_replicator","doc_id":"foo","id":"81cc3633ee8de1332e412ef9052c7b6f","node":"nonode@nohost","source":"foo","target":"bar","state":"crashing","info":"db_not_found: could not open foo","error_count":6,"last_updated":"2017-11-08T18:07:38Z","start_time":"2017-11-08T17:51:52Z","proxy":null},
{"database":"_replicator","doc_id":"foo2","id":null,"source":"http://admin:*****@localhost:5984/foo/","target":"http://admin:*****@localhost:5984/bar/","state":"completed","error_count":0,"info":{"revisions_checked":23,"missing_revisions_found":23,"docs_read":23,"docs_written":23,"changes_pending":null,"doc_write_failures":0,"checkpointed_source_seq":"27-g1AAAAIbeJyV0EsOgjAQBuAGMOLCM-gRSoUKK7mJ9kWQYLtQ13oTvYneRG-CfZAYSUjqZppM5v_SmRYAENchB3OppOKilKpWx1Or2wEBdNF1XVOHJD7oxnTFKMOcDYdH4nSpK930wsQKAmYIVdBXKI2w_RGQyFJYFb7CzgiXXgDuDywXKUk4mJ0lF9VeCj6SlpGu4KofDdyMEFoBk3QtMt87OOXulIdRAqvABHPO0F_K0ymv7zYU5UVe-W_zdoK9R2QFxhjBUAwzzQch86VT"},"start_time":"2017-11-01T21:05:03Z","last_updated":"2017-11-01T21:05:06Z"}
]}`),
			}, nil),
			expected: []*schedulerReplication{
				{
					database:      "_replicator",
					docID:         "foo",
					replicationID: "81cc3633ee8de1332e412ef9052c7b6f",
					state:         "crashing",
					source:        "foo",
					target:        "bar",
					startTime:     parseTime(t, "2017-11-08T17:51:52Z"),
					err: &replicationError{
						status: 404,
						reason: "db_not_found: could not open foo",
					},
				},
				{
					database:  "_replicator",
					docID:     "foo2",
					source:    "http://admin:*****@localhost:5984/foo/",
					target:    "http://admin:*****@localhost:5984/bar/",
					state:     "completed",
					startTime: parseTime(t, "2017-11-01T21:05:03Z"),
					endTime:   parseTime(t, "2017-11-01T21:05:06Z"),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reps, err := test.client.getReplicationsFromScheduler(context.Background(), test.options)
			testy.StatusError(t, test.err, test.status, err)
			result := make([]*schedulerReplication, len(reps))
			for i, rep := range reps {
				result[i] = rep.(*schedulerReplication)
				result[i].db = nil
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}
