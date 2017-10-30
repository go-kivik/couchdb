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
			Name:     "InvalidInput",
			Input:    `"foo"`,
			Error:    `kivik: '"foo"' does not appear to be a valid timestamp`,
			Expected: "0001-01-01 00:00:00 +0000",
		},
	}
	for _, test := range tests {
		func(test stTest) {
			t.Run(test.Name, func(t *testing.T) {
				var result replicationStateTime
				var errMsg string
				if err := json.Unmarshal([]byte(test.Input), &result); err != nil {
					errMsg = err.Error()
				}
				if errMsg != test.Error {
					t.Errorf("Error\nExpected: %s\n  Actual: %s\n", test.Error, errMsg)
				}
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
			status: kivik.StatusInternalServerError,
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

func TestGetReplications(t *testing.T) {
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
			status: kivik.StatusInternalServerError,
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
			reps, err := test.client.GetReplications(context.Background(), test.options)
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
