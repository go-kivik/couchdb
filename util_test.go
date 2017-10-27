package couchdb

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/testy"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		Name     string
		Input    interface{}
		Expected string
		status   int
		err      string
	}{
		{
			Name:     "Null",
			Expected: "null",
		},
		{
			Name:     "String",
			Input:    `{"foo":"bar"}`,
			Expected: `{"foo":"bar"}`,
		},
		{
			Name:     "ByteSlice",
			Input:    []byte(`{"foo":"bar"}`),
			Expected: `{"foo":"bar"}`,
		},
		{
			Name:     "RawMessage",
			Input:    json.RawMessage(`{"foo":"bar"}`),
			Expected: `{"foo":"bar"}`,
		},
		{
			Name:     "Interface",
			Input:    map[string]string{"foo": "bar"},
			Expected: `{"foo":"bar"}`,
		},
		{
			Name:   "invalid json",
			Input:  make(chan int),
			status: kivik.StatusBadRequest,
			err:    "json: unsupported type: chan int",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			r, err := toJSON(test.Input)
			testy.StatusError(t, test.err, test.status, err)
			buf := &bytes.Buffer{}
			buf.ReadFrom(r)
			result := strings.TrimSpace(buf.String())
			if result != test.Expected {
				t.Errorf("Expected: `%s`\n  Actual: `%s`", test.Expected, result)
			}
		})
	}
}
