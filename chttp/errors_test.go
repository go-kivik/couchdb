package chttp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
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
				Response: &http.Response{StatusCode: 400},
			},
			expected: "Bad Request",
		},
		{
			name: "Reason, HTTP code",
			input: &HTTPError{
				Response: &http.Response{StatusCode: 400},
				Reason:   "Bad stuff",
			},
			expected: "Bad Request: Bad stuff",
		},
		{
			name: "Non-HTTP code",
			input: &HTTPError{
				Response: &http.Response{StatusCode: 604},
				Reason:   "Bad stuff",
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
		status   int
		err      string
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
			status: http.StatusNotFound,
			err:    "Not Found",
			expected: &kivik.Error{
				HTTPStatus: http.StatusNotFound,
				FromServer: true,
				Err: &HTTPError{
					Response: &http.Response{
						StatusCode: http.StatusBadRequest,
					},
					exitStatus: ExitNotRetrieved,
				},
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
			status: http.StatusBadRequest,
			err:    "Bad Request: Name: '_foo'. Only lowercase characters (a-z), digits (0-9), and any of the characters _, $, (, ), +, -, and / are allowed. Must begin with a letter.",
			expected: &kivik.Error{
				HTTPStatus: http.StatusBadRequest,
				FromServer: true,
				Err: &HTTPError{
					Response: &http.Response{
						StatusCode: http.StatusBadRequest,
					},
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
			status: http.StatusBadRequest,
			err:    "Bad Request",
			expected: &kivik.Error{
				HTTPStatus: http.StatusBadRequest,
				FromServer: true,
				Err: &HTTPError{
					Response: &http.Response{
						StatusCode: http.StatusBadRequest,
					},
					exitStatus: ExitNotRetrieved,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ResponseError(test.resp)
			testy.StatusError(t, test.err, test.status, err)
			if he, ok := err.(*HTTPError); ok {
				he.Response = nil
				err = he
			}
			if d := diff.Interface(test.expected, err); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestFormatError(t *testing.T) {
	type tst struct {
		err  error
		str  string
		std  string
		full string
	}
	tests := testy.NewTable()
	tests.Add("standard error", tst{
		err:  errors.New("foo"),
		str:  "foo",
		std:  "foo",
		full: "foo",
	})
	tests.Add("HTTPError", tst{
		err: &HTTPError{
			Response: &http.Response{
				StatusCode:    http.StatusNotFound,
				ContentLength: 321,
				Request: &http.Request{
					Method:        http.MethodPost,
					URL:           &url.URL{Scheme: "http", Host: "localhost:5984"},
					ContentLength: 123,
				},
			},
		},
		str: "Not Found",
		std: "Not Found",
		full: `Not Found:
    REQUEST: POST http://localhost:5984 (123 bytes)
    RESPONSE: 404 / Not Found (321 bytes)`,
	})

	tests.Run(t, func(t *testing.T, test tst) {
		if d := diff.Text(test.str, test.err.Error()); d != nil {
			t.Errorf("Error():\n%s", d)
		}
		if d := diff.Text(test.std, fmt.Sprintf("%v", test.err)); d != nil {
			t.Errorf("Standard:\n%s", d)
		}
		if d := diff.Text(test.full, fmt.Sprintf("%+v", test.err)); d != nil {
			t.Errorf("Full:\n%s", d)
		}
	})
}
