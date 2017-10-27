package couchdb

import (
	"encoding/json"
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
