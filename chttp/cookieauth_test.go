package chttp

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"gitlab.com/flimzy/testy"
	"golang.org/x/net/publicsuffix"

	kivik "github.com/go-kivik/kivik/v3"
)

func TestCookieAuthAuthenticate(t *testing.T) {
	type cookieTest struct {
		dsn            string
		auth           *CookieAuth
		err            string
		status         int
		expectedCookie *http.Cookie
	}

	tests := testy.NewTable()
	tests.Add("success", func(t *testing.T) interface{} {
		var sessCounter int
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Type", "application/json")
			h.Set("Date", "Sat, 08 Sep 2018 15:49:29 GMT")
			h.Set("Server", "CouchDB/2.2.0 (Erlang OTP/19)")
			if r.URL.Path == "/_session" {
				sessCounter++
				if sessCounter > 1 {
					t.Fatal("Too many calls to /_session")
				}
				h.Set("Set-Cookie", "AuthSession=YWRtaW46NUI5M0VGODk6eLUGqXf0HRSEV9PPLaZX86sBYes; Version=1; Path=/; HttpOnly")
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"ok":true,"name":"admin","roles":["_admin"]}`))
			} else {
				if cookie := r.Header.Get("Cookie"); cookie != "AuthSession=YWRtaW46NUI5M0VGODk6eLUGqXf0HRSEV9PPLaZX86sBYes" {
					t.Errorf("Expected cookie not found: %s", cookie)
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}
		}))
		return cookieTest{
			dsn:  s.URL,
			auth: &CookieAuth{Username: "foo", Password: "bar"},
			expectedCookie: &http.Cookie{
				Name:  kivik.SessionCookieName,
				Value: "YWRtaW46NUI5M0VGODk6eLUGqXf0HRSEV9PPLaZX86sBYes",
			},
		}
	})
	tests.Add("cookie not set", func(t *testing.T) interface{} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Type", "application/json")
			h.Set("Date", "Sat, 08 Sep 2018 15:49:29 GMT")
			h.Set("Server", "CouchDB/2.2.0 (Erlang OTP/19)")
			w.WriteHeader(200)
		}))
		return cookieTest{
			dsn:  s.URL,
			auth: &CookieAuth{Username: "foo", Password: "bar"},
		}
	})

	tests.Run(t, func(t *testing.T, test cookieTest) {
		c, err := New(test.dsn)
		if err != nil {
			t.Fatal(err)
		}
		if e := c.Auth(test.auth); e != nil {
			t.Fatal(e)
		}
		_, err = c.DoError(context.Background(), "GET", "/foo", nil)
		testy.StatusError(t, test.err, test.status, err)
		if d := testy.DiffInterface(test.expectedCookie, test.auth.Cookie()); d != nil {
			t.Error(d)
		}

		// Do it again; should be idempotent
		_, err = c.DoError(context.Background(), "GET", "/foo", nil)
		testy.StatusError(t, test.err, test.status, err)
		if d := testy.DiffInterface(test.expectedCookie, test.auth.Cookie()); d != nil {
			t.Error(d)
		}
	})
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
			if d := testy.DiffInterface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

type dummyJar []*http.Cookie

var _ http.CookieJar = &dummyJar{}

func (j dummyJar) Cookies(_ *url.URL) []*http.Cookie {
	return []*http.Cookie(j)
}

func (j *dummyJar) SetCookies(_ *url.URL, cookies []*http.Cookie) {
	*j = cookies
}

func Test_shouldAuth(t *testing.T) {
	type tt struct {
		a    *CookieAuth
		req  *http.Request
		want bool
	}

	tests := testy.NewTable()
	tests.Add("no session", tt{
		a:    &CookieAuth{},
		req:  httptest.NewRequest("GET", "/", nil),
		want: true,
	})
	tests.Add("authed request", func() interface{} {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: kivik.SessionCookieName})
		return tt{
			a:    &CookieAuth{},
			req:  req,
			want: false,
		}
	})
	tests.Add("valid session", func() interface{} {
		c, _ := New("http://example.com/")
		c.Jar = &dummyJar{&http.Cookie{
			Name:    kivik.SessionCookieName,
			Expires: time.Now().Add(20 * time.Second),
		}}
		a := &CookieAuth{client: c}

		return tt{
			a:    a,
			req:  httptest.NewRequest("GET", "/", nil),
			want: false,
		}
	})
	tests.Add("expired session", func() interface{} {
		c, _ := New("http://example.com/")
		c.Jar = &dummyJar{&http.Cookie{
			Name:    kivik.SessionCookieName,
			Expires: time.Now().Add(-20 * time.Second),
		}}
		a := &CookieAuth{client: c}

		return tt{
			a:    a,
			req:  httptest.NewRequest("GET", "/", nil),
			want: true,
		}
	})
	tests.Add("no expiry time", func() interface{} {
		c, _ := New("http://example.com/")
		c.Jar = &dummyJar{&http.Cookie{
			Name: kivik.SessionCookieName,
		}}
		a := &CookieAuth{client: c}

		return tt{
			a:    a,
			req:  httptest.NewRequest("GET", "/", nil),
			want: false,
		}
	})

	tests.Run(t, func(t *testing.T, tt tt) {
		got := tt.a.shouldAuth(tt.req)
		if got != tt.want {
			t.Errorf("Want %t, got %t", tt.want, got)
		}
	})
}
