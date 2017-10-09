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
				Request:    &http.Request{Method: http.MethodPost},
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
			err:   "json: cannot unmarshal object into Go struct field queryPlan.fields of type []interface {}",
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
			err:   "json: cannot unmarshal string into Go struct field queryPlan.fields of type []interface {}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(queryPlan)
			err := json.Unmarshal([]byte(test.input), &result)
			testy.Error(t, test.err, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}
