package couchdb

import (
	"encoding/json"
	"testing"

	"github.com/flimzy/testy"
)

func TestSequenceIDUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input string

		expected sequenceID
		err      string
	}{
		{
			name:     "Couch 1.6",
			input:    "123",
			expected: "123",
		},
		{
			name:     "Couch 2.0",
			input:    `"1-seqfoo"`,
			expected: "1-seqfoo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var seq sequenceID
			err := json.Unmarshal([]byte(test.input), &seq)
			testy.Error(t, test.err, err)
			if seq != test.expected {
				t.Errorf("Unexpected result: %s", seq)
			}
		})
	}
}
