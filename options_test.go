package couchdb

import (
	"testing"

	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
)

func TestFullCommit(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
		status   int
		err      string
	}{
		{
			name:     "new",
			input:    map[string]interface{}{OptionFullCommit: true},
			expected: true,
		},
		{
			name:   "new error",
			input:  map[string]interface{}{OptionFullCommit: 123},
			status: kivik.StatusBadRequest,
			err:    "kivik: option 'X-Couch-Full-Commit' must be bool, not int",
		},
		{
			name:     "none",
			input:    nil,
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := fullCommit(test.input)
			testy.StatusError(t, test.err, test.status, err)
			if result != test.expected {
				t.Errorf("Unexpected result: %v", result)
			}
			if _, ok := test.input[OptionFullCommit]; ok {
				t.Errorf("%s still set in options", OptionFullCommit)
			}
		})
	}
}

func TestIfNoneMatch(t *testing.T) {
	tests := []struct {
		name     string
		opts     map[string]interface{}
		expected string
		status   int
		err      string
	}{
		{
			name:     "nil",
			opts:     nil,
			expected: "",
		},
		{
			name:     "inm not set",
			opts:     map[string]interface{}{"foo": "bar"},
			expected: "",
		},
		{
			name:   "wrong type",
			opts:   map[string]interface{}{OptionIfNoneMatch: 123},
			status: kivik.StatusBadRequest,
			err:    "kivik: option 'If-None-Match' must be string, not int",
		},
		{
			name:     "valid",
			opts:     map[string]interface{}{OptionIfNoneMatch: "foo"},
			expected: `"foo"`,
		},
		{
			name:     "valid, pre-quoted",
			opts:     map[string]interface{}{OptionIfNoneMatch: `"foo"`},
			expected: `"foo"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ifNoneMatch(test.opts)
			testy.StatusError(t, test.err, test.status, err)
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
			if _, ok := test.opts[OptionIfNoneMatch]; ok {
				t.Errorf("%s still set in options", OptionIfNoneMatch)
			}
		})
	}
}
