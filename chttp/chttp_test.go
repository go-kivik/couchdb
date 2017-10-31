package chttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/publicsuffix"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/testy"
)

func TestNew(t *testing.T) {
	type newTest struct {
		name     string
		dsn      string
		expected *Client
		status   int
		err      string
	}
	tests := []newTest{
		{
			name:   "invalid url",
			dsn:    "http://foo.com/%xx",
			status: kivik.StatusBadRequest,
			err:    `parse http://foo.com/%xx: invalid URL escape "%xx"`,
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
				w.WriteHeader(kivik.StatusUnauthorized)
			}
			s := httptest.NewServer(http.HandlerFunc(h))
			dsn, _ := url.Parse(s.URL)
			dsn.User = url.UserPassword("user", "password")
			return newTest{
				name:   "auth failed",
				dsn:    dsn.String(),
				status: kivik.StatusUnauthorized,
				err:    "Unauthorized",
			}
		}(),
		func() newTest {
			h := func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(kivik.StatusOK)
				fmt.Fprintf(w, `{"userCtx":{"name":"user"}}`)
			}
			s := httptest.NewServer(http.HandlerFunc(h))
			authDSN, _ := url.Parse(s.URL)
			dsn, _ := url.Parse(s.URL)
			authDSN.User = url.UserPassword("user", "password")
			jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
			return newTest{
				name: "auth success",
				dsn:  authDSN.String(),
				expected: &Client{
					Client: &http.Client{Jar: jar},
					rawDSN: authDSN.String(),
					dsn:    dsn,
					auth: &CookieAuth{
						Username: "user",
						Password: "password",
						dsn:      dsn,
						setJar:   true,
						jar:      jar,
					},
				},
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := New(context.Background(), test.dsn)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
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

func dsn(t *testing.T) string {
	for _, env := range []string{"KIVIK_TEST_DSN_COUCH16", "KIVIK_TEST_DSN_COUCH20", "KIVIK_TEST_DSN_CLOUDANT"} {
		dsn := os.Getenv(env)
		if dsn != "" {
			return dsn
		}
	}
	t.Skip("DSN not set")
	return ""
}

func getClient(t *testing.T) *Client {
	dsn := dsn(t)
	client, err := New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Failed to connect to '%s': %s", dsn, err)
	}
	return client
}

func TestDo(t *testing.T) {
	client := getClient(t)
	res, err := client.DoReq(context.Background(), "GET", "/", &Options{Accept: "application/json"})
	if err != nil {
		t.Errorf("Failed to make request GET /: %s", err)
	}
	body := &bytes.Buffer{}
	if _, err = body.ReadFrom(res.Body); err != nil {
		t.Errorf("Failed to read response body: %s", err)
	}
	if !strings.Contains(body.String(), `"couchdb"`) {
		t.Errorf("Body does not contain `\"couchdb\"` as expected: `%s`", body)
	}
	var i interface{}
	if err = json.Unmarshal(body.Bytes(), &i); err != nil {
		t.Errorf("Body is not valid JSON: %s", err)
	}
	ct, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil {
		t.Errorf("Failed to parse content type: %s", err)
	}
	if ct != "application/json" {
		t.Errorf("Unexpected content type: %s", ct)
	}
}

func TestJSONBody(t *testing.T) {
	client := getClient(t)
	bogusQuery := map[string]string{
		"foo": "bar",
		"bar": "baz",
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(bogusQuery); err != nil {
		t.Fatalf("JSON encoding failed: %s", err)
	}
	_, err := client.DoReq(context.Background(), "POST", "/_session", &Options{Body: buf})
	if err != nil {
		t.Errorf("Failed to make request: %s", err)
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
		Name     string
		Input    interface{}
		Error    string
		Expected string
	}
	tests := []encodeTest{
		{
			Name:     "Null",
			Expected: "null",
		},
		{
			Name: "Struct",
			Input: struct {
				Foo string `json:"foo"`
			}{Foo: "bar"},
			Expected: `{"foo":"bar"}`,
		},
		{
			Name:  "JSONError",
			Input: func() {}, // Functions cannot be marshaled to JSON
			Error: "json: unsupported type: func()",
		},
	}
	for _, test := range tests {
		func(test encodeTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				r, errFunc := EncodeBody(test.Input, func() {})
				buf := &bytes.Buffer{}
				_, _ = buf.ReadFrom(r)
				var msg string
				if err := errFunc(); err != nil {
					msg = err.Error()
				}
				result := strings.TrimSpace(buf.String())
				if result != test.Expected {
					t.Errorf("Result\nExpected: %s\n  Actual: %s\n", test.Expected, result)
				}
				if msg != test.Error {
					t.Errorf("Error\nExpected: %s\n  Actual: %s\n", test.Error, msg)
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
				"Accept":       []string{"application/json"},
				"Content-Type": []string{"application/json"},
			},
		},
		{
			Name:    "Content-Type",
			Options: &Options{ContentType: "image/gif"},
			Expected: http.Header{
				"Accept":       []string{"application/json"},
				"Content-Type": []string{"image/gif"},
			},
		},
		{
			Name:    "Accept",
			Options: &Options{Accept: "image/gif"},
			Expected: http.Header{
				"Accept":       []string{"image/gif"},
				"Content-Type": []string{"application/json"},
			},
		},
		{
			Name:    "ForceCommit",
			Options: &Options{ForceCommit: true},
			Expected: http.Header{
				"Accept":              []string{"application/json"},
				"Content-Type":        []string{"application/json"},
				"X-Couch-Full-Commit": []string{"true"},
			},
		},
		{
			Name:    "Destination",
			Options: &Options{Destination: "somewhere nice"},
			Expected: http.Header{
				"Accept":       []string{"application/json"},
				"Content-Type": []string{"application/json"},
				"Destination":  []string{"somewhere nice"},
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
				Body:       ioutil.NopCloser(strings.NewReader("")),
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
