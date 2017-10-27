package couchdb

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestDeJSONify(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		status   int
		err      string
	}{
		{
			name:     "string",
			input:    `{"foo":"bar"}`,
			expected: map[string]interface{}{"foo": "bar"},
		},
		{
			name:     "[]byte",
			input:    []byte(`{"foo":"bar"}`),
			expected: map[string]interface{}{"foo": "bar"},
		},
		{
			name:     "json.RawMessage",
			input:    json.RawMessage(`{"foo":"bar"}`),
			expected: map[string]interface{}{"foo": "bar"},
		},
		{
			name:     "map",
			input:    map[string]string{"foo": "bar"},
			expected: map[string]string{"foo": "bar"},
		},
		{
			name:   "invalid JSON sring",
			input:  `{"foo":"\C"}`,
			status: kivik.StatusBadRequest,
			err:    "invalid character 'C' in string escape code",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := deJSONify(test.input)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestExplain(t *testing.T) {
	tests := []struct {
		name     string
		db       *db
		query    interface{}
		expected *driver.QueryPlan
		status   int
		err      string
	}{
		{
			name: "CouchDB 1.6",
			db: &db{
				client: &client{Compat: CompatCouch16},
			},
			status: kivik.StatusNotImplemented,
			err:    "kivik: Find interface not implemented prior to CouchDB 2.0.0",
		},
		{
			name:   "invalid query",
			db:     &db{client: &client{}},
			query:  make(chan int),
			status: kivik.StatusInternalServerError,
			err:    "json: unsupported type: chan int",
		},
		{
			name:   "transport error",
			db:     newTestDB(nil, errors.New("xport error")),
			status: kivik.StatusInternalServerError,
			err:    "Post http://example.com/testdb/_explain: xport error",
		},
		{
			name: "db error",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusNotFound,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusNotFound,
			err:    "Not Found",
		},
		{
			name: "success",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"dbname":"foo"}`)),
			}, nil),
			expected: &driver.QueryPlan{DBName: "foo"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Explain(context.Background(), test.query)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestUnmarshalQueryPlan(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *queryPlan
		err      string
	}{
		{
			name:  "non-array",
			input: `{"fields":{}}`,
			err:   "json: cannot unmarshal object into Go",
		},
		{
			name:     "all_fields",
			input:    `{"fields":"all_fields","dbname":"foo"}`,
			expected: &queryPlan{DBName: "foo"},
		},
		{
			name:     "simple field list",
			input:    `{"fields":["foo","bar"],"dbname":"foo"}`,
			expected: &queryPlan{Fields: []interface{}{"foo", "bar"}, DBName: "foo"},
		},
		{
			name:  "complex field list",
			input: `{"dbname":"foo", "fields":[{"foo":"asc"},{"bar":"desc"}]}`,
			expected: &queryPlan{DBName: "foo",
				Fields: []interface{}{map[string]interface{}{"foo": "asc"},
					map[string]interface{}{"bar": "desc"}}},
		},
		{
			name:  "invalid bare string",
			input: `{"fields":"not_all_fields"}`,
			err:   "json: cannot unmarshal string into Go",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(queryPlan)
			err := json.Unmarshal([]byte(test.input), &result)
			testy.ErrorRE(t, test.err, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestCreateIndex(t *testing.T) {
	tests := []struct {
		testName   string
		ddoc, name string
		index      interface{}
		db         *db
		status     int
		err        string
	}{
		{
			name:   "Couch 1.6",
			db:     &db{client: &client{Compat: CompatCouch16}},
			status: kivik.StatusNotImplemented,
			err:    "kivik: Find interface not implemented prior to CouchDB 2.0.0",
		},
		{
			name:   "invalid JSON index",
			db:     newTestDB(nil, nil),
			index:  `invalid json`,
			status: kivik.StatusBadRequest,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name:   "invalid raw index",
			db:     newTestDB(nil, nil),
			index:  map[string]interface{}{"foo": make(chan int)},
			status: kivik.StatusBadRequest,
			err:    "json: unsupported type: chan int",
		},
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: 500,
			err:    "Post http://example.com/testdb/_index: net error",
		},
		{
			name: "success 2.1.0",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"X-CouchDB-Body-Time": {"0"},
					"X-Couch-Request-ID":  {"8e4aef0c2f"},
					"Server":              {"CouchDB/2.1.0 (Erlang OTP/17)"},
					"Date":                {"Fri, 27 Oct 2017 18:14:38 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"126"},
					"Cache-Control":       {"must-revalidate"},
				},
				Body: Body(`{"result":"created","id":"_design/a7ee061f1a2c0c6882258b2f1e148b714e79ccea","name":"a7ee061f1a2c0c6882258b2f1e148b714e79ccea"}`),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			err := test.db.CreateIndex(context.Background(), test.ddoc, test.name, test.index)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}
