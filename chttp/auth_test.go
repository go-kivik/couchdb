package chttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/flimzy/testy"
	"golang.org/x/net/publicsuffix"

	"github.com/go-kivik/kivik"
)

type mockRT struct {
	resp *http.Response
	err  error
}

var _ http.RoundTripper = &mockRT{}

func (rt *mockRT) RoundTrip(_ *http.Request) (*http.Response, error) {
	return rt.resp, rt.err
}

func TestAuthenticate(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close() // nolint: errcheck
		var authed bool
		if auth := r.Header.Get("Authorization"); auth == "Basic YWRtaW46YWJjMTIz" {
			authed = true
		}
		if r.Method == http.MethodPost {
			var result struct {
				Name     string
				Password string
			}
			_ = json.NewDecoder(r.Body).Decode(&result)
			if result.Name == "admin" && result.Password == "abc123" {
				authed = true
				http.SetCookie(w, &http.Cookie{
					Name:     kivik.SessionCookieName,
					Value:    "auth-token",
					Path:     "/",
					HttpOnly: true,
				})
			}
		}
		if ses := r.Header.Get("Cookie"); ses == "AuthSession=auth-token" {
			authed = true
		}
		if !authed {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/_session" { // nolint: goconst
			_, _ = w.Write([]byte(`{"userCtx":{"name":"admin"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"foo":123}`))
	}))

	type authTest struct {
		addr       string
		jar        http.CookieJar
		auther     Authenticator
		authErr    string
		authStatus int
		err        string
		status     int
	}
	tests := testy.NewTable()
	tests.Cleanup(s.Close)
	tests.Add("unauthorized", authTest{
		addr:   s.URL,
		err:    "Unauthorized",
		status: http.StatusUnauthorized,
	})
	tests.Add("basic auth", authTest{
		addr:   s.URL,
		auther: &BasicAuth{Username: "admin", Password: "abc123"},
	})
	tests.Add("cookie auth success", func(t *testing.T) interface{} {
		sv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Type", "application/json")
			h.Set("Date", "Sat, 08 Sep 2018 15:49:29 GMT")
			h.Set("Server", "CouchDB/2.2.0 (Erlang OTP/19)")
			if r.URL.Path == "/_session" {
				h.Set("Set-Cookie", "AuthSession=YWRtaW46NUI5M0VGODk6eLUGqXf0HRSEV9PPLaZX86sBYes; Version=1; Path=/; HttpOnly")
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"ok":true,"name":"admin","roles":["_admin"]}`))
			} else {
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}
		}))
		return authTest{
			addr:   sv.URL,
			auther: &CookieAuth{Username: "foo", Password: "bar"},
		}
	})
	tests.Add("failed basic auth", authTest{
		addr:   s.URL,
		auther: &BasicAuth{Username: "foo"},
		err:    "Unauthorized",
		status: http.StatusUnauthorized,
	})
	tests.Add("failed cookie auth", authTest{
		addr:   s.URL,
		auther: &CookieAuth{Username: "foo"},
		err:    "Get " + s.URL + "/foo: Unauthorized",
		status: http.StatusUnauthorized,
	})
	tests.Add("already authenticated with cookie", func() interface{} {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			t.Fatal(err)
		}
		u, _ := url.Parse(s.URL)
		jar.SetCookies(u, []*http.Cookie{{
			Name:     kivik.SessionCookieName,
			Value:    "auth-token",
			Path:     "/",
			HttpOnly: true,
		}})
		return authTest{
			addr: s.URL,
			jar:  jar,
		}
	})

	tests.Run(t, func(t *testing.T, test authTest) {
		ctx := context.Background()
		c, err := New(test.addr)
		if err != nil {
			t.Fatal(err)
		}
		if test.jar != nil {
			c.Client.Jar = test.jar
		}
		if test.auther != nil {
			e := c.Auth(test.auther)
			testy.StatusError(t, test.authErr, test.authStatus, e)
		}
		_, err = c.DoError(ctx, "GET", "/foo", nil)
		testy.StatusError(t, test.err, test.status, err)
	})
}
