package chttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"golang.org/x/net/publicsuffix"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/errors"
)

var defaultUA = func() string {
	c := &Client{}
	return c.userAgent()
}()

func TestNew(t *testing.T) {
	type newTest struct {
		name       string
		dsn        string
		expected   *Client
		status     int
		curlStatus int
		err        string
	}
	tests := []newTest{
		{
			name:       "invalid url",
			dsn:        "http://foo.com/%xx",
			status:     kivik.StatusBadAPICall,
			curlStatus: ExitStatusURLMalformed,
			err:        `parse http://foo.com/%xx: invalid URL escape "%xx"`,
		},
		{
			name:       "no url",
			dsn:        "",
			status:     kivik.StatusBadAPICall,
			curlStatus: ExitFailedToInitialize,
			err:        "no URL specified",
		},
		{
			name: "no auth",
			dsn:  "http://foo.com/",
			expected: &Client{
				Client: &http.Client{},
				rawDSN: "http://foo.com/",
				dsn: &url.URL{
					Scheme: "http",
					Host:   "foo.com",
					Path:   "/",
				},
			},
		},
		func() newTest {
			h := func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(kivik.StatusOK)
				fmt.Fprintf(w, `{"userCtx":{"name":"user"}}`) // nolint: errcheck
			}
			s := httptest.NewServer(http.HandlerFunc(h))
			authDSN, _ := url.Parse(s.URL)
			dsn, _ := url.Parse(s.URL + "/")
			authDSN.User = url.UserPassword("user", "password")
			jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
			c := &Client{
				Client: &http.Client{Jar: jar},
				rawDSN: authDSN.String(),
				dsn:    dsn,
			}
			auth := &CookieAuth{
				Username:  "user",
				Password:  "password",
				client:    c,
				transport: http.DefaultTransport,
			}
			c.auth = auth
			c.Client.Transport = auth
			return newTest{
				name:     "auth success",
				dsn:      authDSN.String(),
				expected: c,
			}
		}(),
		{
			name: "default url scheme",
			dsn:  "foo.com",
			expected: &Client{
				Client: &http.Client{},
				rawDSN: "foo.com",
				dsn: &url.URL{
					Scheme: "http",
					Host:   "foo.com",
					Path:   "/",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := New(test.dsn)
			curlStatusErrorRE(t, test.err, test.status, test.curlStatus, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expected           *url.URL
		status, curlStatus int
		err                string
	}{
		{
			name:  "happy path",
			input: "http://foo.com/",
			expected: &url.URL{
				Scheme: "http",
				Host:   "foo.com",
				Path:   "/",
			},
		},
		{
			name:  "default scheme",
			input: "foo.com",
			expected: &url.URL{
				Scheme: "http",
				Host:   "foo.com",
				Path:   "/",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parseDSN(test.input)
			curlStatusErrorRE(t, test.err, test.status, test.curlStatus, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Fatal(d)
			}
		})
	}
}

func TestDSN(t *testing.T) {
	expected := "foo"
	client := &Client{rawDSN: expected}
	result := client.DSN()
	if result != expected {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestFixPath(t *testing.T) {
	tests := []struct {
		Input    string
		Expected string
	}{
		{Input: "foo", Expected: "/foo"},
		{Input: "foo?oink=yes", Expected: "/foo"},
		{Input: "foo/bar", Expected: "/foo/bar"},
		{Input: "foo%2Fbar", Expected: "/foo%2Fbar"},
	}
	for _, test := range tests {
		req, _ := http.NewRequest("GET", "http://localhost/"+test.Input, nil)
		fixPath(req, test.Input)
		if req.URL.EscapedPath() != test.Expected {
			t.Errorf("Path for '%s' not fixed.\n\tExpected: %s\n\t  Actual: %s\n", test.Input, test.Expected, req.URL.EscapedPath())
		}
	}
}

func TestEncodeBody(t *testing.T) {
	type encodeTest struct {
		name  string
		input interface{}

		expected string
		status   int
		err      string
	}
	tests := []encodeTest{
		{
			name:     "Null",
			input:    nil,
			expected: "null",
		},
		{
			name: "Struct",
			input: struct {
				Foo string `json:"foo"`
			}{Foo: "bar"},
			expected: `{"foo":"bar"}`,
		},
		{
			name:   "JSONError",
			input:  func() {}, // Functions cannot be marshaled to JSON
			status: http.StatusBadRequest,
			err:    "json: unsupported type: func()",
		},
		{
			name:     "raw json input",
			input:    json.RawMessage(`{"foo":"bar"}`),
			expected: `{"foo":"bar"}`,
		},
		{
			name:     "byte slice input",
			input:    []byte(`{"foo":"bar"}`),
			expected: `{"foo":"bar"}`,
		},
		{
			name:     "string input",
			input:    `{"foo":"bar"}`,
			expected: `{"foo":"bar"}`,
		},
	}
	for _, test := range tests {
		func(test encodeTest) {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()
				r := EncodeBody(test.input)
				defer r.Close() // nolint: errcheck
				body, err := ioutil.ReadAll(r)
				testy.StatusError(t, test.err, test.status, err)
				result := strings.TrimSpace(string(body))
				if result != test.expected {
					t.Errorf("Result\nExpected: %s\n  Actual: %s\n", test.expected, result)
				}
			})
		}(test)
	}
}

func TestSetHeaders(t *testing.T) {
	type shTest struct {
		Name     string
		Options  *Options
		Expected http.Header
	}
	tests := []shTest{
		{
			Name: "NoOpts",
			Expected: http.Header{
				"Accept":       {"application/json"},
				"Content-Type": {"application/json"},
			},
		},
		{
			Name:    "Content-Type",
			Options: &Options{ContentType: "image/gif"},
			Expected: http.Header{
				"Accept":       {"application/json"},
				"Content-Type": {"image/gif"},
			},
		},
		{
			Name:    "Accept",
			Options: &Options{Accept: "image/gif"},
			Expected: http.Header{
				"Accept":       {"image/gif"},
				"Content-Type": {"application/json"},
			},
		},
		{
			Name:    "FullCommit",
			Options: &Options{FullCommit: true},
			Expected: http.Header{
				"Accept":              {"application/json"},
				"Content-Type":        {"application/json"},
				"X-Couch-Full-Commit": {"true"},
			},
		},
		{
			Name:    "Destination",
			Options: &Options{Destination: "somewhere nice"},
			Expected: http.Header{
				"Accept":       {"application/json"},
				"Content-Type": {"application/json"},
				"Destination":  {"somewhere nice"},
			},
		},
		{
			Name:    "If-None-Match",
			Options: &Options{IfNoneMatch: `"foo"`},
			Expected: http.Header{
				"Accept":        {"application/json"},
				"Content-Type":  {"application/json"},
				"If-None-Match": {`"foo"`},
			},
		},
		{
			Name:    "Unquoted If-None-Match",
			Options: &Options{IfNoneMatch: `foo`},
			Expected: http.Header{
				"Accept":        {"application/json"},
				"Content-Type":  {"application/json"},
				"If-None-Match": {`"foo"`},
			},
		},
	}
	for _, test := range tests {
		func(test shTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					panic(err)
				}
				setHeaders(req, test.Options)
				if d := diff.Interface(test.Expected, req.Header); d != nil {
					t.Errorf("Headers:\n%s\n", d)
				}
			})
		}(test)
	}
}

func TestSetQuery(t *testing.T) {
	tests := []struct {
		name     string
		req      *http.Request
		opts     *Options
		expected *http.Request
	}{
		{
			name:     "nil query",
			req:      &http.Request{URL: &url.URL{}},
			expected: &http.Request{URL: &url.URL{}},
		},
		{
			name:     "empty query",
			req:      &http.Request{URL: &url.URL{RawQuery: "a=b"}},
			opts:     &Options{Query: url.Values{}},
			expected: &http.Request{URL: &url.URL{RawQuery: "a=b"}},
		},
		{
			name:     "options query",
			req:      &http.Request{URL: &url.URL{}},
			opts:     &Options{Query: url.Values{"foo": []string{"a"}}},
			expected: &http.Request{URL: &url.URL{RawQuery: "foo=a"}},
		},
		{
			name:     "merged queries",
			req:      &http.Request{URL: &url.URL{RawQuery: "bar=b"}},
			opts:     &Options{Query: url.Values{"foo": []string{"a"}}},
			expected: &http.Request{URL: &url.URL{RawQuery: "bar=b&foo=a"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setQuery(test.req, test.opts)
			if d := diff.Interface(test.expected, test.req); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestETag(t *testing.T) {
	tests := []struct {
		name     string
		input    *http.Response
		expected string
		found    bool
	}{
		{
			name:     "nil response",
			input:    nil,
			expected: "",
			found:    false,
		},
		{
			name:     "No etag",
			input:    &http.Response{},
			expected: "",
			found:    false,
		},
		{
			name: "ETag",
			input: &http.Response{
				Header: http.Header{
					"ETag": {`"foo"`},
				},
			},
			expected: "foo",
			found:    true,
		},
		{
			name: "Etag",
			input: &http.Response{
				Header: http.Header{
					"Etag": {`"bar"`},
				},
			},
			expected: "bar",
			found:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, found := ETag(test.input)
			if result != test.expected {
				t.Errorf("Unexpected result: %s", result)
			}
			if found != test.found {
				t.Errorf("Unexpected found: %v", found)
			}
		})
	}
}

func TestGetRev(t *testing.T) {
	tests := []struct {
		name          string
		resp          *http.Response
		expected, err string
	}{
		{
			name: "error response",
			resp: &http.Response{
				StatusCode: 400,
				Request:    &http.Request{Method: "POST"},
				Body:       ioutil.NopCloser(strings.NewReader("")),
			},
			err: "Bad Request",
		},
		{
			name: "no ETag header",
			resp: &http.Response{
				StatusCode: 200,
				Request:    &http.Request{Method: "POST"},
				Body:       ioutil.NopCloser(strings.NewReader("")),
			},
			err: "no ETag header found",
		},
		{
			name: "normalized Etag header",
			resp: &http.Response{
				StatusCode: 200,
				Request:    &http.Request{Method: "POST"},
				Header:     http.Header{"Etag": {`"12345"`}},
				Body:       ioutil.NopCloser(strings.NewReader("")),
			},
			expected: `12345`,
		},
		{
			name: "satndard ETag header",
			resp: &http.Response{
				StatusCode: 200,
				Request:    &http.Request{Method: "POST"},
				Header:     http.Header{"ETag": {`"12345"`}},
				Body:       Body(""),
			},
			expected: `12345`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := GetRev(test.resp)
			testy.Error(t, test.err, err)
			if result != test.expected {
				t.Errorf("Got %s, expected %s", result, test.expected)
			}
		})
	}
}

func TestDoJSON(t *testing.T) {
	tests := []struct {
		name         string
		method, path string
		opts         *Options
		client       *Client
		expected     interface{}
		response     *http.Response
		status       int
		err          string
	}{
		{
			name:   "network error",
			method: "GET",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com: net error",
		},
		{
			name:   "error response",
			method: "GET",
			client: newTestClient(&http.Response{
				StatusCode: 401,
				Header: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"67"},
				},
				ContentLength: 67,
				Body:          Body(`{"error":"unauthorized","reason":"Name or password is incorrect."}`),
				Request:       &http.Request{Method: "GET"},
			}, nil),
			status: kivik.StatusUnauthorized,
			err:    "Unauthorized: Name or password is incorrect.",
		},
		{
			name:   "invalid JSON in response",
			method: "GET",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"67"},
				},
				ContentLength: 67,
				Body:          Body(`invalid response`),
				Request:       &http.Request{Method: "GET"},
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name:   "success",
			method: "GET",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"15"},
				},
				ContentLength: 15,
				Body:          Body(`{"foo":"bar"}`),
				Request:       &http.Request{Method: "GET"},
			}, nil),
			expected: map[string]interface{}{"foo": "bar"},
			response: &http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"15"},
				},
				ContentLength: 15,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var i interface{}
			response, err := test.client.DoJSON(context.Background(), test.method, test.path, test.opts, &i)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, i); d != nil {
				t.Errorf("JSON result differs:\n%s\n", d)
			}
			response.Request = nil
			response.Body = nil
			if d := diff.Interface(test.response, response); d != nil {
				t.Errorf("Response differs:\n%s\n", d)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name               string
		method, path       string
		body               io.Reader
		expected           *http.Request
		client             *Client
		status, curlStatus int
		err                string
	}{
		{
			name:       "invalid URL",
			client:     newTestClient(nil, nil),
			method:     "GET",
			path:       "%xx",
			status:     kivik.StatusBadAPICall,
			curlStatus: ExitStatusURLMalformed,
			err:        `parse %xx: invalid URL escape "%xx"`,
		},
		{
			name:       "invalid method",
			method:     "FOO BAR",
			client:     newTestClient(nil, nil),
			status:     kivik.StatusBadAPICall,
			curlStatus: 0,
			err:        `net/http: invalid method "FOO BAR"`,
		},
		{
			name:   "success",
			method: "GET",
			path:   "foo",
			client: newTestClient(nil, nil),
			expected: &http.Request{
				Method: "GET",
				URL: func() *url.URL {
					url := newTestClient(nil, nil).dsn
					url.Path = "/foo"
					return url
				}(),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header: http.Header{
					"User-Agent": []string{defaultUA},
				},
				Host: "example.com",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := test.client.NewRequest(context.Background(), test.method, test.path, test.body)
			curlStatusErrorRE(t, test.err, test.status, test.curlStatus, err)
			test.expected = test.expected.WithContext(req.Context()) // determinism
			if d := diff.Interface(test.expected, req); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDoReq(t *testing.T) {
	tests := []struct {
		name               string
		trace              func(t *testing.T, success *bool) *ClientTrace
		method, path       string
		opts               *Options
		client             *Client
		status, curlStatus int
		err                string
	}{
		{
			name:       "no method",
			status:     500,
			curlStatus: 0,
			err:        "chttp: method required",
		},
		{
			name:       "invalid url",
			method:     "GET",
			path:       "%xx",
			client:     newTestClient(nil, nil),
			status:     kivik.StatusBadAPICall,
			curlStatus: ExitStatusURLMalformed,
			err:        `parse %xx: invalid URL escape "%xx"`,
		},
		{
			name:   "network error",
			method: "GET",
			path:   "foo",
			client: newTestClient(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/foo: net error",
		},
		{
			name:   "error response",
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: 400,
				Body:       Body(""),
			}, nil),
			// No error here
		},
		{
			name:   "success",
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Body:       Body(""),
			}, nil),
			// success!
		},
		{
			name:   "body error",
			method: "PUT",
			path:   "foo",
			client: newTestClient(nil, errors.Status(kivik.StatusBadRequest, "bad request")),
			status: kivik.StatusBadRequest,
			err:    "Put http://example.com/foo: bad request",
		},
		{
			name: "response trace",
			trace: func(t *testing.T, success *bool) *ClientTrace {
				return &ClientTrace{
					HTTPResponse: func(r *http.Response) {
						*success = true
						expected := &http.Response{StatusCode: 200}
						if d := diff.HTTPResponse(expected, r); d != nil {
							t.Error(d)
						}
					},
				}
			},
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Body:       Body(""),
			}, nil),
			// response body trace
		},
		{
			name: "response body trace",
			trace: func(t *testing.T, success *bool) *ClientTrace {
				return &ClientTrace{
					HTTPResponseBody: func(r *http.Response) {
						*success = true
						expected := &http.Response{
							StatusCode: 200,
							Body:       Body("foo"),
						}
						if d := diff.HTTPResponse(expected, r); d != nil {
							t.Error(d)
						}
					},
				}
			},
			method: "PUT",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Body:       Body("foo"),
			}, nil),
			// response trace
		},
		{
			name: "request trace",
			trace: func(t *testing.T, success *bool) *ClientTrace {
				return &ClientTrace{
					HTTPRequest: func(r *http.Request) {
						*success = true
						expected := httptest.NewRequest("PUT", "/foo", nil)
						expected.Header.Add("Accept", "application/json")
						expected.Header.Add("Content-Type", "application/json")
						expected.Header.Add("User-Agent", defaultUA)
						if d := diff.HTTPRequest(expected, r); d != nil {
							t.Error(d)
						}
					},
				}
			},
			method: "PUT",
			path:   "/foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Body:       Body("foo"),
			}, nil),
			opts: &Options{
				Body: Body("bar"),
			},
			// request trace
		},
		{
			name: "request body trace",
			trace: func(t *testing.T, success *bool) *ClientTrace {
				return &ClientTrace{
					HTTPRequestBody: func(r *http.Request) {
						*success = true
						expected := httptest.NewRequest("PUT", "/foo", Body("bar"))
						expected.Header.Add("Accept", "application/json")
						expected.Header.Add("Content-Type", "application/json")
						expected.Header.Add("User-Agent", defaultUA)
						if d := diff.HTTPRequest(expected, r); d != nil {
							t.Error(d)
						}
					},
				}
			},
			method: "PUT",
			path:   "/foo",
			client: newTestClient(&http.Response{
				StatusCode: 200,
				Body:       Body("foo"),
			}, nil),
			opts: &Options{
				Body: Body("bar"),
			},
			// request body trace
		},
		{
			name: "couchdb mounted below root",
			client: newCustomClient("http://foo.com/dbroot/", func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/dbroot/foo" {
					return nil, errors.Errorf("Unexpected path: %s", r.URL.Path)
				}
				return &http.Response{}, nil
			}),
			method: "GET",
			path:   "/foo",
		},
		{
			name: "user agent",
			client: newCustomClient("http://foo.com/", func(r *http.Request) (*http.Response, error) {
				if ua := r.UserAgent(); ua != defaultUA {
					return nil, errors.Errorf("Unexpected User Agent: %s", ua)
				}
				return &http.Response{}, nil
			}),
			method: "GET",
			path:   "/foo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			traceSuccess := true
			if test.trace != nil {
				traceSuccess = false
				ctx = WithClientTrace(ctx, test.trace(t, &traceSuccess))
			}
			_, err := test.client.DoReq(ctx, test.method, test.path, test.opts)
			curlStatusErrorRE(t, test.err, test.status, test.curlStatus, err)
			if !traceSuccess {
				t.Error("Trace failed")
			}
		})
	}
}

func TestDoError(t *testing.T) {
	tests := []struct {
		name         string
		method, path string
		opts         *Options
		client       *Client
		status       int
		err          string
	}{
		{
			name:   "no method",
			status: 500,
			err:    "chttp: method required",
		},
		{
			name:   "error response",
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       Body(""),
				Request:    &http.Request{Method: "GET"},
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name:   "success",
			method: "GET",
			path:   "foo",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       Body(""),
				Request:    &http.Request{Method: "GET"},
			}, nil),
			// No error
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.client.DoError(context.Background(), test.method, test.path, test.opts)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestNetError(t *testing.T) {
	tests := []struct {
		name  string
		input error

		status, curlStatus int
		err                string
	}{
		{
			name:       "nil",
			input:      nil,
			status:     0,
			curlStatus: 0,
			err:        "",
		},
		{
			name: "timeout",
			input: func() error {
				s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
					time.Sleep(1 * time.Second)
				}))
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
				defer cancel()
				req, err := http.NewRequest("GET", s.URL, nil)
				if err != nil {
					t.Fatal(err)
				}
				_, err = http.DefaultClient.Do(req.WithContext(ctx))
				return err
			}(),
			status:     kivik.StatusNetworkError,
			curlStatus: ExitOperationTimeout,
			err:        `Get http://127.0.0.1:\d+: context deadline exceeded`,
		},
		{
			name: "cannot resolve host",
			input: func() error {
				req, err := http.NewRequest("GET", "http://foo.com.invalid.hostname", nil)
				if err != nil {
					t.Fatal(err)
				}
				_, err = http.DefaultClient.Do(req)
				return err
			}(),
			status:     kivik.StatusNetworkError,
			curlStatus: ExitHostNotResolved,
			err:        ": no such host$",
		},
		{
			name: "connection refused",
			input: func() error {
				req, err := http.NewRequest("GET", "http://localhost:99", nil)
				if err != nil {
					t.Fatal(err)
				}
				_, err = http.DefaultClient.Do(req)
				return err
			}(),
			status:     kivik.StatusNetworkError,
			curlStatus: ExitFailedToConnect,
			err:        ": connection refused$",
		},
		{
			name: "too many redirects",
			input: func() error {
				var s *httptest.Server
				redirHandler := func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, s.URL, 302)
				}
				s = httptest.NewServer(http.HandlerFunc(redirHandler))
				_, err := http.Get(s.URL)
				return err
			}(),
			status:     kivik.StatusNetworkError,
			curlStatus: ExitTooManyRedirects,
			err:        `^Get http://127.0.0.1:\d+: stopped after 10 redirects$`,
		},
		{
			name: "url error",
			input: &url.Error{
				Op:  "Get",
				URL: "http://foo.com/",
				Err: errors.New("some error"),
			},
			status: kivik.StatusNetworkError,
			// curlStatus: ExitStatusURLMalformed,
			err: "Get http://foo.com/: some error",
		},
		{
			name: "url error with embedded status",
			input: &url.Error{
				Op:  "Get",
				URL: "http://foo.com/",
				Err: errors.Status(kivik.StatusBadRequest, "some error"),
			},
			status: kivik.StatusBadRequest,
			err:    "Get http://foo.com/: some error",
		},
		{
			name:       "other error",
			input:      errors.New("other error"),
			status:     kivik.StatusNetworkError,
			curlStatus: ExitUnknownFailure,
			err:        "other error",
		},
		{
			name:       "other error with embedded status",
			input:      errors.Status(kivik.StatusBadRequest, "bad req"),
			status:     kivik.StatusBadRequest,
			curlStatus: 0,
			err:        "bad req",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := netError(test.input)
			curlStatusErrorRE(t, test.err, test.status, test.curlStatus, err)
		})
	}
}

func TestUserAgent(t *testing.T) {
	tests := []struct {
		name     string
		ua       []string
		expected string
	}{
		{
			name: "defaults",
			expected: fmt.Sprintf("%s/%s (Language=%s; Platform=%s/%s)",
				UserAgent, Version, runtime.Version(), runtime.GOARCH, runtime.GOOS),
		},
		{
			name: "custom",
			ua:   []string{"Oinky/1.2.3"},
			expected: fmt.Sprintf("%s/%s (Language=%s; Platform=%s/%s) Oinky/1.2.3",
				UserAgent, Version, runtime.Version(), runtime.GOARCH, runtime.GOOS),
		},
		{
			name: "multiple",
			ua:   []string{"Oinky/1.2.3", "Moo/5.4.3"},
			expected: fmt.Sprintf("%s/%s (Language=%s; Platform=%s/%s) Oinky/1.2.3 Moo/5.4.3",
				UserAgent, Version, runtime.Version(), runtime.GOARCH, runtime.GOOS),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Client{
				UserAgents: test.ua,
			}
			result := c.userAgent()
			if result != test.expected {
				t.Errorf("Unexpected user agent: %s", result)
			}
		})
	}
}
