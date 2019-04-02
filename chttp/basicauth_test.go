package chttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flimzy/diff"
)

func TestBasicAuthRoundTrip(t *testing.T) {
	type rtTest struct {
		name     string
		auth     *BasicAuth
		req      *http.Request
		expected *http.Response
		cleanup  func()
	}
	tests := []rtTest{
		{
			name: "Provided transport",
			req:  httptest.NewRequest("GET", "/", nil),
			auth: &BasicAuth{
				Username: "foo",
				Password: "bar",
				transport: customTransport(func(req *http.Request) (*http.Response, error) {
					u, p, ok := req.BasicAuth()
					if !ok {
						t.Error("BasicAuth not set in request")
					}
					if u != "foo" || p != "bar" { // nolint: goconst
						t.Errorf("Unexpected user/password: %s/%s", u, p)
					}
					return &http.Response{StatusCode: 200}, nil
				}),
			},
			expected: &http.Response{StatusCode: 200},
		},
		func() rtTest {
			h := func(w http.ResponseWriter, r *http.Request) {
				u, p, ok := r.BasicAuth()
				if !ok {
					t.Error("BasicAuth not set in request")
				}
				if u != "foo" || p != "bar" {
					t.Errorf("Unexpected user/password: %s/%s", u, p)
				}
				w.Header().Set("Date", "Wed, 01 Nov 2017 19:32:41 GMT")
				w.Header().Set("Content-Type", "application/json")
			}
			s := httptest.NewServer(http.HandlerFunc(h))
			return rtTest{
				name: "default transport",
				auth: &BasicAuth{
					Username:  "foo",
					Password:  "bar",
					transport: http.DefaultTransport,
				},
				req: httptest.NewRequest("GET", s.URL, nil),
				expected: &http.Response{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header: http.Header{
						"Content-Length": {"0"},
						"Content-Type":   {"application/json"},
						"Date":           {"Wed, 01 Nov 2017 19:32:41 GMT"},
					},
				},
				cleanup: func() { s.Close() },
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := test.auth.RoundTrip(test.req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body = nil
			res.Request = nil
			if d := diff.Interface(test.expected, res); d != nil {
				t.Error(d)
			}
		})
	}
}
