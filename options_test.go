package couchdb

import (
	"testing"

	"github.com/flimzy/testy"
)

func TestFullCommit(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
		err      string
	}{
		{
			name:     "legacy",
			input:    map[string]interface{}{"force_commit": true},
			expected: true,
		},
		{
			name:  "legacy error",
			input: map[string]interface{}{"force_commit": 123},
			err:   "kivik: option 'force_commit' must be bool, not int",
		},
		{
			name:     "new",
			input:    map[string]interface{}{"full-commit": true},
			expected: true,
		},
		{
			name:  "new error",
			input: map[string]interface{}{"full-commit": 123},
			err:   "kivik: option 'full-commit' must be bool, not int",
		},
		{
			name: "new priority over old",
			input: map[string]interface{}{
				"full-commit":  false,
				"force_commit": true,
			},
			expected: false,
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
			testy.Error(t, test.err, err)
			if result != test.expected {
				t.Errorf("Unexpected result: %v", result)
			}
		})
	}
}
