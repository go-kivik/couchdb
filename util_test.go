package couchdb

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
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
