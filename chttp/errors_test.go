package chttp

import (
	"net/http"
	"testing"

	"github.com/flimzy/diff"
	"github.com/go-kivik/kivik"
)

func TestHTTPErrorError(t *testing.T) {
	tests := []struct {
		name     string
		input    *HTTPError
		expected string
	}{
		{
			name: "No reason",
			input: &HTTPError{
				Code: 400,
			},
			expected: "Bad Request",
		},
		{
			name: "Reason, HTTP code",
			input: &HTTPError{
				Code:   400,
				Reason: "Bad stuff",
			},
			expected: "Bad Request: Bad stuff",
		},
		{
			name: "Non-HTTP code",
			input: &HTTPError{
				Code:   604,
				Reason: "Bad stuff",
			},
			expected: "Bad stuff",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.input.Error()
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	}
}

func TestResponseError(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		expected interface{}
	}{
		{
			name:     "non error",
			resp:     &http.Response{StatusCode: 200},
			expected: nil,
		},
		{
			name: "HEAD error",
			resp: &http.Response{
				StatusCode: http.StatusNotFound,
				Request:    &http.Request{Method: "HEAD"},
				Body:       Body(""),
			},
			expected: &kivik.Error{
				HTTPStatus: http.StatusNotFound,
				FromServer: true,
				Err:        &HTTPError{Code: http.StatusNotFound, exitStatus: ExitNotRetrieved},
			},
		},
		{
			name: "2.0.0 error",
			resp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Cache-Control":       {"must-revalidate"},
					"Content-Length":      {"194"},
					"Content-Type":        {"application/json"},
					"Date":                {"Fri, 27 Oct 2017 15:34:07 GMT"},
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"X-Couch-Request-ID":  {"92d05bd015"},
					"X-CouchDB-Body-Time": {"0"},
				},
				ContentLength: 194,
				Body:          Body(`{"error":"illegal_database_name","reason":"Name: '_foo'. Only lowercase characters (a-z), digits (0-9), and any of the characters _, $, (, ), +, -, and / are allowed. Must begin with a letter."}`),
				Request:       &http.Request{Method: "PUT"},
			},
			expected: &kivik.Error{
				HTTPStatus: http.StatusBadRequest,
				FromServer: true,
				Err: &HTTPError{
					Code:       http.StatusBadRequest,
					exitStatus: ExitNotRetrieved,
					Reason:     "Name: '_foo'. Only lowercase characters (a-z), digits (0-9), and any of the characters _, $, (, ), +, -, and / are allowed. Must begin with a letter.",
				},
			},
		},
		{
			name: "invalid json error",
			resp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Fri, 27 Oct 2017 15:42:34 GMT"},
					"Content-Type":   {"application/json"},
					"Content-Length": {"194"},
					"Cache-Control":  {"must-revalidate"},
				},
				ContentLength: 194,
				Body:          Body("invalid json"),
				Request:       &http.Request{Method: "PUT"},
			},
			expected: &kivik.Error{
				HTTPStatus: http.StatusBadRequest,
				FromServer: true,
				Err:        &HTTPError{Code: http.StatusBadRequest, exitStatus: ExitNotRetrieved},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ResponseError(test.resp)
			if d := diff.Interface(test.expected, err); d != nil {
				t.Error(d)
			}
		})
	}
}
