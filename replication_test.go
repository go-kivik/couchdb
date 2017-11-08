package couchdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestStateTime(t *testing.T) {
	type stTest struct {
		Name     string
		Input    string
		Error    string
		Expected string
	}
	tests := []stTest{
		{
			Name:     "Blank",
			Error:    "unexpected end of JSON input",
			Expected: "0001-01-01 00:00:00 +0000",
		},
		{
			Name:     "ValidRFC3339",
			Input:    `"2011-02-17T20:22:02+01:00"`,
			Expected: "2011-02-17 20:22:02 +0100",
		},
		{
			Name:     "ValidUnixTimestamp",
			Input:    "1492543959",
			Expected: "2017-04-18 19:32:39 +0000",
		},
		{
			Name:     "invalid timestamp",
			Input:    `"foo"`,
			Error:    `kivik: '"foo"' does not appear to be a valid timestamp`,
			Expected: "0001-01-01 00:00:00 +0000",
		},
	}
	for _, test := range tests {
		func(test stTest) {
			t.Run(test.Name, func(t *testing.T) {
				var result replicationStateTime
				err := json.Unmarshal([]byte(test.Input), &result)
				testy.Error(t, test.Error, err)
				if r := time.Time(result).Format("2006-01-02 15:04:05 -0700"); r != test.Expected {
					t.Errorf("Result\nExpected: %s\n  Actual: %s\n", test.Expected, r)
				}
			})
		}(test)
	}
}

func TestReplicationErrorUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *replicationError
		err      string
	}{
		{
			name:  "doc example 1",
			input: `"db_not_found: could not open http://adm:*****@localhost:5984/missing/"`,
			expected: &replicationError{
				status: kivik.StatusNotFound,
				reason: "db_not_found: could not open http://adm:*****@localhost:5984/missing/",
			},
		},
		{
			name:  "timeout",
			input: `"timeout: some timeout occurred"`,
			expected: &replicationError{
				status: kivik.StatusRequestTimeout,
				reason: "timeout: some timeout occurred",
			},
		},
		{
			name:  "unknown",
			input: `"unknown error"`,
			expected: &replicationError{
				status: kivik.StatusInternalServerError,
				reason: "unknown error",
			},
		},
		{
			name:  "invalid JSON",
			input: `"\C"`,
			err:   "invalid character 'C' in string escape code",
		},
		{
			name:  "Unauthorized",
			input: `"unauthorized: unauthorized to access or create database foo"`,
			expected: &replicationError{
				status: kivik.StatusUnauthorized,
				reason: "unauthorized: unauthorized to access or create database foo",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repErr := new(replicationError)
			err := repErr.UnmarshalJSON([]byte(test.input))
			testy.Error(t, test.err, err)
			if d := diff.Interface(test.expected, repErr); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestReplicate(t *testing.T) {
	tests := []struct {
		name           string
		target, source string
		options        map[string]interface{}
		client         *client
		status         int
		err            string
	}{
		{
			name:   "no target",
			status: kivik.StatusBadRequest,
			err:    "kivik: targetDSN required",
		},
		{
			name:   "no source",
			target: "foo",
			status: kivik.StatusBadRequest,
			err:    "kivik: sourceDSN required",
		},
		{
			name:   "invalid options",
			target: "foo", source: "bar",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "json: unsupported type: chan int",
		},
		{
			name:   "network error",
			target: "foo", source: "bar",
			client: newTestClient(nil, errors.New("net eror")),
			status: kivik.StatusNetworkError,
			err:    "Post http://example.com/_replicator: net eror",
		},
		{
			name:   "1.6.1",
			target: "foo", source: "bar",
			client: newCustomClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 201,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Location":       {"http://localhost:5984/_replicator/4ab99e4d7d4b5a6c5a6df0d0ed01221d"},
						"ETag":           {`"1-290800e5803500237075f9b08226cffd"`},
						"Date":           {"Mon, 30 Oct 2017 20:03:34 GMT"},
						"Content-Type":   {"application/json"},
						"Content-Length": {"95"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"ok":true,"id":"4ab99e4d7d4b5a6c5a6df0d0ed01221d","rev":"1-290800e5803500237075f9b08226cffd"}`),
				}, nil
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, err := test.client.Replicate(context.Background(), test.target, test.source, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := resp.(*replication); !ok {
				t.Errorf("Unexpected response type: %T", resp)
			}
		})
	}
}

type replicationRow struct {
	ReplicationID string
	Source        string
	Target        string
	StartTime     time.Time
	EndTime       time.Time
	State         string
	Status        int
	Err           string
}

func TestGetReplicationsFromScheduler(t *testing.T) {
	tests := []struct {
		name     string
		options  map[string]interface{}
		client   *client
		expected []replicationRow
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
			expected: []replicationRow{
				{
					ReplicationID: "81cc3633ee8de1332e412ef9052c7b6f",
					Source:        "foo",
					Target:        "bar",
					StartTime:     parseTime(t, "2017-11-08T17:51:52Z"),
					State:         "crashing",
				},
				{
					Source:    "http://admin:*****@localhost:5984/foo/",
					Target:    "http://admin:*****@localhost:5984/bar/",
					StartTime: parseTime(t, "2017-11-01T21:05:03Z"),
					EndTime:   parseTime(t, "2017-11-01T21:05:06Z"),
					State:     "completed",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reps, err := test.client.getReplicationsFromScheduler(context.Background(), test.options)
			testy.StatusError(t, test.err, test.status, err)
			result := make([]replicationRow, len(reps))
			for i, rep := range reps {
				var msg string
				if e := rep.Err(); e != nil {
					msg = e.Error()
				}
				result[i] = replicationRow{
					ReplicationID: rep.ReplicationID(),
					Source:        rep.Source(),
					Target:        rep.Target(),
					StartTime:     rep.StartTime(),
					EndTime:       rep.EndTime(),
					State:         rep.State(),
					Status:        kivik.StatusCode(rep.Err()),
					Err:           msg,
				}
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestLegacyGetReplications(t *testing.T) {
	tests := []struct {
		name     string
		options  map[string]interface{}
		client   *client
		expected []replicationRow
		status   int
		err      string
	}{
		{
			name:    "invalid options",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name:   "network error",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_replicator/_all_docs?include_docs=true: net error",
		},
		{
			name: "success, 1.6.1",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Transfer-Encoding": {"chunked"},
					"Server":            {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"ETag":              {`"97AGDUD7SV24L2PLSG3XG4MOY"`},
					"Date":              {"Mon, 30 Oct 2017 20:31:31 GMT"},
					"Content-Type":      {"application/json"},
					"Cache-Control":     {"must-revalidate"},
				},
				Body: Body(`{"total_rows":2,"offset":0,"rows":[
				{"id":"4ab99e4d7d4b5a6c5a6df0d0ed01221d","key":"4ab99e4d7d4b5a6c5a6df0d0ed01221d","value":{"rev":"2-6419706e969050d8000efad07259de4f"},"doc":{"_id":"4ab99e4d7d4b5a6c5a6df0d0ed01221d","_rev":"2-6419706e969050d8000efad07259de4f","source":"foo","target":"bar","owner":"admin","_replication_state":"error","_replication_state_time":"2017-10-30T20:03:34+00:00","_replication_state_reason":"unauthorized: unauthorized to access or create database foo","_replication_id":"548507fbb9fb9fcd8a3b27050b9ba5bf"}},
				{"id":"_design/_replicator","key":"_design/_replicator","value":{"rev":"1-5bfa2c99eefe2b2eb4962db50aa3cfd4"},"doc":{"_id":"_design/_replicator","_rev":"1-5bfa2c99eefe2b2eb4962db50aa3cfd4","language":"javascript","validate_doc_update":"..."}}
				]}`),
			}, nil),
			expected: []replicationRow{
				{
					ReplicationID: "548507fbb9fb9fcd8a3b27050b9ba5bf",
					Source:        "foo",
					Target:        "bar",
					State:         "error",
					Status:        kivik.StatusUnauthorized,
					EndTime:       parseTime(t, "2017-10-30T20:03:34+00:00"),
					Err:           "unauthorized: unauthorized to access or create database foo",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reps, err := test.client.legacyGetReplications(context.Background(), test.options)
			testy.StatusError(t, test.err, test.status, err)
			result := make([]replicationRow, len(reps))
			for i, rep := range reps {
				var msg string
				if e := rep.Err(); e != nil {
					msg = e.Error()
				}
				result[i] = replicationRow{
					ReplicationID: rep.ReplicationID(),
					Source:        rep.Source(),
					Target:        rep.Target(),
					StartTime:     rep.StartTime(),
					EndTime:       rep.EndTime(),
					State:         rep.State(),
					Status:        kivik.StatusCode(rep.Err()),
					Err:           msg,
				}
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestGetReplications(t *testing.T) {
	tests := []struct {
		name        string
		client      *client
		status      int
		err         string
		noScheduler bool
	}{
		{
			name:        "network error",
			client:      newTestClient(nil, errors.New("net error")),
			noScheduler: false,
			status:      kivik.StatusNetworkError,
			err:         "Get http://example.com/_scheduler/docs: net error",
		},
		{
			name: "not found, 2.0",
			client: newTestClient(&http.Response{
				StatusCode: 404,
				Request:    &http.Request{Method: "GET"},
				Body:       Body(""),
			}, nil),
			noScheduler: true,
			status:      kivik.StatusNotFound,
			err:         "Not Found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.client.GetReplications(context.Background(), nil)
			if test.client.noScheduler != test.noScheduler {
				t.Errorf("Unexpected noScheduler state: %t", test.client.noScheduler)
			}
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestReplicationUpdate(t *testing.T) {
	tests := []struct {
		name     string
		rep      *replication
		expected *driver.ReplicationInfo
		status   int
		err      string
	}{
		{
			name: "network error",
			rep: &replication{
				docID: "4ab99e4d7d4b5a6c5a6df0d0ed01221d",
				db:    newTestDB(nil, errors.New("net error")),
			},
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb/4ab99e4d7d4b5a6c5a6df0d0ed01221d: net error",
		},
		{
			name: "no active reps 1.6.1",
			rep: &replication{
				docID: "4ab99e4d7d4b5a6c5a6df0d0ed01221d",
				db: newCustomDB(func(req *http.Request) (*http.Response, error) {
					switch req.URL.Path {
					case "/testdb/4ab99e4d7d4b5a6c5a6df0d0ed01221d":
						return &http.Response{
							StatusCode: 200,
							Header: http.Header{
								"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
								"ETag":           {`"2-6419706e969050d8000efad07259de4f"`},
								"Date":           {"Mon, 30 Oct 2017 20:57:15 GMT"},
								"Content-Type":   {"application/json"},
								"Content-Length": {"359"},
								"Cache-Control":  {"must-revalidate"},
							},
							Body: Body(`{"_id":"4ab99e4d7d4b5a6c5a6df0d0ed01221d","_rev":"2-6419706e969050d8000efad07259de4f","source":"foo","target":"bar","owner":"admin","_replication_state":"error","_replication_state_time":"2017-10-30T20:03:34+00:00","_replication_state_reason":"unauthorized: unauthorized to access or create database foo","_replication_id":"548507fbb9fb9fcd8a3b27050b9ba5bf"}`),
						}, nil
					case "/_active_tasks":
						return &http.Response{
							StatusCode: 200,
							Header: http.Header{
								"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
								"Date":           {"Mon, 30 Oct 2017 21:06:40 GMT"},
								"Content-Type":   {"application/json"},
								"Content-Length": {"3"},
								"Cache-Control":  {"must-revalidate"},
							},
							Body: Body(`[]`),
						}, nil
					default:
						panic("Unknown req path: " + req.URL.Path)
					}
				}),
			},
			expected: &driver.ReplicationInfo{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(driver.ReplicationInfo)
			err := test.rep.Update(context.Background(), result)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestReplicationDelete(t *testing.T) {
	tests := []struct {
		name   string
		rep    *replication
		status int
		err    string
	}{
		{
			name: "network error",
			rep: &replication{
				docID: "foo",
				db:    newTestDB(nil, errors.New("net error")),
			},
			status: kivik.StatusNetworkError,
			err:    "Head http://example.com/testdb/foo: net error",
		},
		{
			name: "delete network error",
			rep: &replication{
				docID: "4ab99e4d7d4b5a6c5a6df0d0ed01221d",
				db: newCustomDB(func(req *http.Request) (*http.Response, error) {
					if req.Method == "HEAD" {
						return &http.Response{
							StatusCode: 200,
							Header: http.Header{
								"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
								"ETag":           {`"2-6419706e969050d8000efad07259de4f"`},
								"Date":           {"Mon, 30 Oct 2017 21:14:46 GMT"},
								"Content-Type":   {"application/json"},
								"Content-Length": {"359"},
								"Cache-Control":  {"must-revalidate"},
							},
							Body: Body(""),
						}, nil
					}
					return nil, errors.New("delete error")
				}),
			},
			status: kivik.StatusNetworkError,
			err:    "^(Delete http://example.com/testdb/4ab99e4d7d4b5a6c5a6df0d0ed01221d\\?rev=2-6419706e969050d8000efad07259de4f: )?delete error",
		},
		{
			name: "success, 1.6.1",
			rep: &replication{
				docID: "4ab99e4d7d4b5a6c5a6df0d0ed01221d",
				db: newCustomDB(func(req *http.Request) (*http.Response, error) {
					if req.Method == "HEAD" {
						return &http.Response{
							StatusCode: 200,
							Header: http.Header{
								"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
								"ETag":           {`"2-6419706e969050d8000efad07259de4f"`},
								"Date":           {"Mon, 30 Oct 2017 21:14:46 GMT"},
								"Content-Type":   {"application/json"},
								"Content-Length": {"359"},
								"Cache-Control":  {"must-revalidate"},
							},
							Body: Body(""),
						}, nil
					}
					return &http.Response{
						StatusCode: 200,
						Header: http.Header{
							"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
							"ETag":           {`"3-2ae9fa6e1f8982a08c4a42b3943e67c5"`},
							"Date":           {"Mon, 30 Oct 2017 21:29:43 GMT"},
							"Content-Type":   {"application/json"},
							"Content-Length": {"95"},
							"Cache-Control":  {"must-revalidate"},
						},
						Body: Body(`{"ok":true,"id":"4ab99e4d7d4b5a6c5a6df0d0ed01221d","rev":"3-2ae9fa6e1f8982a08c4a42b3943e67c5"}`),
					}, nil
				}),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.rep.Delete(context.Background())
			testy.StatusErrorRE(t, test.err, test.status, err)
		})
	}
}

func TestUpdateActiveTasks(t *testing.T) {
	tests := []struct {
		name     string
		rep      *replication
		expected *activeTask
		status   int
		err      string
	}{
		{
			name: "network error",
			rep: &replication{
				db: newTestDB(nil, errors.New("net error")),
			},
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/_active_tasks: net error",
		},
		{
			name: "error response",
			rep: &replication{
				db: newTestDB(&http.Response{
					StatusCode: 500,
					Request:    &http.Request{Method: "GET"},
					Body:       Body(""),
				}, nil),
			},
			status: kivik.StatusInternalServerError,
			err:    "Internal Server Error",
		},
		{
			name: "invalid json response",
			rep: &replication{
				db: newTestDB(&http.Response{
					StatusCode: 200,
					Body:       Body("invalid json"),
				}, nil),
			},
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "rep not found",
			rep: &replication{
				replicationID: "foo",
				db: newTestDB(&http.Response{
					StatusCode: 200,
					Body:       Body("[]"),
				}, nil),
			},
			status: kivik.StatusNotFound,
			err:    "task not found",
		},
		{
			name: "rep found",
			rep: &replication{
				replicationID: "foo",
				db: newTestDB(&http.Response{
					StatusCode: 200,
					Body: Body(`[
						{"type":"foo"},
						{"type":"replication","replication_id":"unf"},
						{"type":"replication","replication_id":"foo","docs_written":1}
					]`),
				}, nil),
			},
			expected: &activeTask{
				Type:          "replication",
				ReplicationID: "foo",
				DocsWritten:   1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.rep.updateActiveTasks(context.Background())
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestSetFromReplicatorDoc(t *testing.T) {
	tests := []struct {
		name     string
		rep      *replication
		doc      *replicatorDoc
		expected *replication
	}{
		{
			name: "started",
			rep:  &replication{},
			doc: &replicatorDoc{
				State:     string(kivik.ReplicationStarted),
				StateTime: replicationStateTime(parseTime(t, "2017-01-01T01:01:01Z")),
			},
			expected: &replication{
				state:     "triggered",
				startTime: parseTime(t, "2017-01-01T01:01:01Z"),
			},
		},
		{
			name: "errored",
			rep:  &replication{},
			doc: &replicatorDoc{
				State:     string(kivik.ReplicationError),
				StateTime: replicationStateTime(parseTime(t, "2017-01-01T01:01:01Z")),
			},
			expected: &replication{
				state:   "error",
				endTime: parseTime(t, "2017-01-01T01:01:01Z"),
			},
		},
		{
			name: "completed",
			rep:  &replication{},
			doc: &replicatorDoc{
				State:     string(kivik.ReplicationComplete),
				StateTime: replicationStateTime(parseTime(t, "2017-01-01T01:01:01Z")),
			},
			expected: &replication{
				state:   "completed",
				endTime: parseTime(t, "2017-01-01T01:01:01Z"),
			},
		},
		{
			name: "set fields",
			rep:  &replication{},
			doc: &replicatorDoc{
				Source:        "foo",
				Target:        "bar",
				ReplicationID: "oink",
				Error:         &replicationError{status: 500, reason: "unf"},
			},
			expected: &replication{
				source:        "foo",
				target:        "bar",
				replicationID: "oink",
				err:           &replicationError{status: 500, reason: "unf"},
			},
		},
		{
			name: "validate that existing fields aren't re-set",
			rep:  &replication{source: "a", target: "b", replicationID: "c", err: errors.New("foo")},
			doc: &replicatorDoc{
				Source:        "foo",
				Target:        "bar",
				ReplicationID: "oink",
			},
			expected: &replication{
				source:        "a",
				target:        "b",
				replicationID: "c",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.rep.setFromReplicatorDoc(test.doc)
			if d := diff.Interface(test.expected, test.rep); d != nil {
				t.Error(d)
			}
		})
	}
}
