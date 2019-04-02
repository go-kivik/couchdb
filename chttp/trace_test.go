package chttp

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
)

func TestHTTPResponse(t *testing.T) {
	tests := []struct {
		name      string
		trace     func(t *testing.T) *ClientTrace
		resp      *http.Response
		finalResp *http.Response
	}{
		{
			name:      "no hook defined",
			trace:     func(_ *testing.T) *ClientTrace { return &ClientTrace{} },
			resp:      &http.Response{StatusCode: 200},
			finalResp: &http.Response{StatusCode: 200},
		},
		{
			name: "HTTPResponseBody/cloned response",
			trace: func(t *testing.T) *ClientTrace {
				return &ClientTrace{
					HTTPResponseBody: func(r *http.Response) {
						if r.StatusCode != 200 {
							t.Errorf("Unexpected status code: %d", r.StatusCode)
						}
						r.StatusCode = 0
						defer r.Body.Close() // nolint: errcheck
						if _, err := ioutil.ReadAll(r.Body); err != nil {
							t.Fatal(err)
						}
					},
				}
			},
			resp:      &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader("testing"))},
			finalResp: &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader("testing"))},
		},
		{
			name: "HTTPResponse/cloned response",
			trace: func(t *testing.T) *ClientTrace {
				return &ClientTrace{
					HTTPResponse: func(r *http.Response) {
						if r.StatusCode != 200 {
							t.Errorf("Unexpected status code: %d", r.StatusCode)
						}
						r.StatusCode = 0
						if r.Body != nil {
							t.Errorf("non-nil body")
						}
					},
				}
			},
			resp:      &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader("testing"))},
			finalResp: &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader("testing"))},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			trace := test.trace(t)
			trace.httpResponseBody(test.resp)
			trace.httpResponse(test.resp)
			if d := diff.HTTPResponse(test.finalResp, test.resp); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestHTTPRequest(t *testing.T) {
	tests := []struct {
		name     string
		trace    func(t *testing.T) *ClientTrace
		req      *http.Request
		finalReq *http.Request
	}{
		{
			name:     "no hook defined",
			trace:    func(_ *testing.T) *ClientTrace { return &ClientTrace{} },
			req:      httptest.NewRequest("PUT", "/", ioutil.NopCloser(strings.NewReader("testing"))),
			finalReq: httptest.NewRequest("PUT", "/", ioutil.NopCloser(strings.NewReader("testing"))),
		},
		{
			name: "HTTPRequesteBody/cloned response",
			trace: func(t *testing.T) *ClientTrace {
				return &ClientTrace{
					HTTPRequestBody: func(r *http.Request) {
						if r.Method != "PUT" {
							t.Errorf("Unexpected method: %s", r.Method)
						}
						r.Method = "unf"     // nolint: goconst
						defer r.Body.Close() // nolint: errcheck
						if _, err := ioutil.ReadAll(r.Body); err != nil {
							t.Fatal(err)
						}
					},
				}
			},
			req:      httptest.NewRequest("PUT", "/", ioutil.NopCloser(strings.NewReader("testing"))),
			finalReq: httptest.NewRequest("PUT", "/", ioutil.NopCloser(strings.NewReader("testing"))),
		},
		{
			name: "HTTPRequeste/cloned response",
			trace: func(t *testing.T) *ClientTrace {
				return &ClientTrace{
					HTTPRequest: func(r *http.Request) {
						if r.Method != "PUT" {
							t.Errorf("Unexpected method: %s", r.Method)
						}
						r.Method = "unf"
						if r.Body != nil {
							t.Errorf("non-nil body")
						}
					},
				}
			},
			req:      httptest.NewRequest("PUT", "/", ioutil.NopCloser(strings.NewReader("testing"))),
			finalReq: httptest.NewRequest("PUT", "/", ioutil.NopCloser(strings.NewReader("testing"))),
		},
		{
			name: "HTTPRequesteBody/no body",
			trace: func(t *testing.T) *ClientTrace {
				return &ClientTrace{
					HTTPRequestBody: func(r *http.Request) {
						if r.Method != "GET" {
							t.Errorf("Unexpected method: %s", r.Method)
						}
						r.Method = "unf"
						if r.Body != nil {
							t.Errorf("non-nil body")
						}
					},
				}
			},
			req: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				return req
			}(),
			finalReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Add("Host", "example.com")
				return req
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			trace := test.trace(t)
			trace.httpRequestBody(test.req)
			trace.httpRequest(test.req)
			if d := diff.HTTPRequest(test.finalReq, test.req); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestReplayReadCloser(t *testing.T) {
	tests := []struct {
		name     string
		input    io.ReadCloser
		expected string
		readErr  string
		closeErr string
	}{
		{
			name:     "no errors",
			input:    ioutil.NopCloser(strings.NewReader("testing")),
			expected: "testing",
		},
		{
			name:     "read error",
			input:    ioutil.NopCloser(&errReader{Reader: strings.NewReader("testi"), err: errors.New("read error 1")}),
			expected: "testi",
			readErr:  "read error 1",
		},
		{
			name:     "close error",
			input:    &errCloser{Reader: strings.NewReader("testin"), err: errors.New("close error 1")},
			expected: "testin",
			closeErr: "close error 1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content, err := ioutil.ReadAll(test.input.(io.Reader))
			closeErr := test.input.Close()
			rc := newReplay(content, err, closeErr)

			result, resultErr := ioutil.ReadAll(rc.(io.Reader))
			resultCloseErr := rc.Close()
			if d := diff.Text(test.expected, result); d != nil {
				t.Error(d)
			}
			testy.Error(t, test.readErr, resultErr)
			testy.Error(t, test.closeErr, resultCloseErr)
		})
	}
}
