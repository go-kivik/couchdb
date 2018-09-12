package chttp

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"github.com/go-kivik/kivik"
	"golang.org/x/net/publicsuffix"
)

func TestCookieAuthAuthenticate(t *testing.T) {
	tests := []struct {
		name           string
		auth           *CookieAuth
		client         *Client
		status         int
		err            string
		expectedCookie *http.Cookie
	}{
		{
			name: "standard request",
			auth: &CookieAuth{
				Username: "foo",
				Password: "bar",
			},
			client: &Client{
				Client: &http.Client{
					Transport: &mockRT{
						resp: &http.Response{
							Header: http.Header{
								"Set-Cookie": []string{
									"AuthSession=cm9vdDo1MEJCRkYwMjq0LO0ylOIwShrgt8y-UkhI-c6BGw; Version=1; Path=/; HttpOnly",
								},
							},
							Body: ioutil.NopCloser(strings.NewReader(`{"userCtx":{"name":"foo"}}`)),
						},
					},
				},
				dsn: &url.URL{Scheme: "http", Host: "foo.com"},
			},
			expectedCookie: &http.Cookie{
				Name:  kivik.SessionCookieName,
				Value: "cm9vdDo1MEJCRkYwMjq0LO0ylOIwShrgt8y-UkhI-c6BGw",
			},
		},
		{
			name: "Invalid JSON response",
			auth: &CookieAuth{
				Username: "foo",
				Password: "bar",
			},
			client: &Client{
				Client: &http.Client{
					Jar: &cookiejar.Jar{},
					Transport: &mockRT{
						resp: &http.Response{
							Body: ioutil.NopCloser(strings.NewReader(`{"asdf"}`)),
						},
					},
				},
				dsn: &url.URL{Scheme: "http", Host: "foo.com"},
			},
			status: kivik.StatusBadResponse,
			err:    "invalid character '}' after object key",
		},
		{
			name: "names don't match",
			auth: &CookieAuth{
				Username: "foo",
				Password: "bar",
			},
			client: &Client{
				Client: &http.Client{
					Transport: &mockRT{
						resp: &http.Response{
							Header: http.Header{
								"Set-Cookie": []string{
									"AuthSession=cm9vdDo1MEJCRkYwMjq0LO0ylOIwShrgt8y-UkhI-c6BGw; Version=1; Path=/; HttpOnly",
								},
							},
							Body: ioutil.NopCloser(strings.NewReader(`{"userCtx":{"name":"notfoo"}}`)),
						},
					},
				},
				dsn: &url.URL{Scheme: "http", Host: "foo.com"},
			},
			status: kivik.StatusBadResponse,
			err:    "auth response for unexpected user",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.auth.Authenticate(context.Background(), test.client)
			testy.StatusError(t, test.err, test.status, err)
			cookie := test.auth.Cookie()
			if d := diff.Interface(test.expectedCookie, cookie); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestCookie(t *testing.T) {
	tests := []struct {
		name     string
		auth     *CookieAuth
		expected *http.Cookie
	}{
		{
			name:     "No cookie jar",
			auth:     &CookieAuth{},
			expected: nil,
		},
		{
			name:     "No dsn",
			auth:     &CookieAuth{},
			expected: nil,
		},
		{
			name:     "no cookies",
			auth:     &CookieAuth{},
			expected: nil,
		},
		{
			name: "cookie found",
			auth: func() *CookieAuth {
				dsn, err := url.Parse("http://example.com/")
				if err != nil {
					t.Fatal(err)
				}
				jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
				if err != nil {
					t.Fatal(err)
				}
				jar.SetCookies(dsn, []*http.Cookie{
					{Name: kivik.SessionCookieName, Value: "foo"},
					{Name: "other", Value: "bar"},
				})
				return &CookieAuth{
					client: &Client{
						dsn: dsn,
						Client: &http.Client{
							Jar: jar,
						},
					},
				}
			}(),
			expected: &http.Cookie{Name: kivik.SessionCookieName, Value: "foo"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.auth.Cookie()
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}
