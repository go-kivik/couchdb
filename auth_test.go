package couchdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flimzy/testy"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
)

type mockAuther struct {
	authCalls int
	authErr   error
}

var _ chttp.Authenticator = &mockAuther{}

func (a *mockAuther) Authenticate(c *chttp.Client) error {
	a.authCalls++
	return a.authErr
}

func (a *mockAuther) Logout(ctx context.Context, c *chttp.Client) error {
	return nil
}

func (a *mockAuther) Check() error {
	if a.authCalls == 1 {
		return nil
	}
	return fmt.Errorf("auth called %d times", a.authCalls)
}

type checker interface {
	Check() error
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name          string
		client        *client
		authenticator interface{}
		status        int
		err           string
	}{
		{
			name:          "invalid authenticator",
			authenticator: 1,
			status:        http.StatusBadRequest,
			err:           "kivik: invalid authenticator",
		},
		{
			name:          "valid authenticator",
			client:        &client{Client: &chttp.Client{}},
			authenticator: &mockAuther{},
		},
		{
			name:          "auth failure",
			client:        &client{Client: &chttp.Client{}},
			authenticator: &mockAuther{authErr: &kivik.Error{HTTPStatus: http.StatusUnauthorized, Err: errors.New("auth failed")}},
			status:        http.StatusUnauthorized,
			err:           "auth failed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.client.Authenticate(context.Background(), test.authenticator)
			testy.StatusError(t, test.err, test.status, err)
			if c, ok := test.authenticator.(checker); ok {
				if e := c.Check(); e != nil {
					t.Error(e)
				}
			}
		})
	}
}

func TestAuthentication(t *testing.T) {
	type tst struct {
		handler    func(*testing.T) http.Handler
		setup      func(*testing.T, *client)
		auther     Authenticator
		authStatus int
		authErr    string
		status     int
		err        string
	}

	handler200 := func(_ *testing.T) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
		})
	}
	tests := testy.NewTable()
	tests.Add("SetTransport", tst{
		handler: handler200,
		auther: SetTransport(customTransport(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("transport error")
		})),
		status: kivik.StatusNetworkError,
		err:    "transport error",
	})
	tests.Add("SetTransport again", tst{
		handler: handler200,
		auther: SetTransport(customTransport(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("transport error")
		})),
		setup: func(t *testing.T, c *client) {
			c.Client.Client.Transport = http.DefaultTransport
		},
		authStatus: kivik.StatusBadAPICall,
		authErr:    "kivik: HTTP client transport already set",
	})
	tests.Add("BasicAuth", tst{
		handler: func(t *testing.T) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if h := r.Header.Get("Authorization"); h != "Basic Ym9iOmFiYzEyMw==" {
					t.Errorf("Unexpected Auth header: %s\n", h)
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{}`))
			})
		},
		auther: BasicAuth("bob", "abc123"),
	})
	tests.Add("CookieAuth", tst{
		handler: func(t *testing.T) http.Handler {
			expectedPaths := []string{"/_session", "/"}
			i := -1
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				i++
				if p := r.URL.Path; p != expectedPaths[i] {
					t.Errorf("Unexpected path: %s\n", p)
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{}`))
			})
		},
		auther: CookieAuth("bob", "abc123"),
	})
	tests.Add("ProxyAuth", tst{
		handler: func(t *testing.T) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if h := r.Header.Get("X-Auth-CouchDB-UserName"); h != "bob" {
					t.Errorf("Unexpected X-Auth-CouchDB-UserName header: %s", h)
				}
				if h := r.Header.Get("X-Auth-CouchDB-Roles"); h != "users,admins" {
					t.Errorf("Unexpected X-Auth-CouchDB-Roles header: %s", h)
				}
				if h := r.Header.Get("Moo"); h != "adedb8d002eb53a52faba80e82cb1fc6d57bca74" {
					t.Errorf("Token header override failed: %s instead of 'adedb8d002eb53a52faba80e82cb1fc6d57bca74'", h)
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{}`))
			})
		},
		auther: ProxyAuth("bob", "abc123", []string{"users", "admins"}, map[string]string{"X-Auth-CouchDB-Token": "moo"}),
	})
	tests.Add("SetCookie", tst{
		handler: func(t *testing.T) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := r.Cookie("cow")
				if err != nil {
					t.Fatal(err)
				}
				if c.Value != "moo" {
					t.Errorf("Unexpected cookie value: %s\n", c.Value)
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{}`))
			})
		},
		auther: SetCookie(&http.Cookie{Name: "cow", Value: "moo"}),
	})
	tests.Add("SetCookie again", tst{
		handler: func(t *testing.T) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := r.Cookie("cow")
				if err != nil {
					t.Fatal(err)
				}
				if c.Value != "moo" {
					t.Errorf("Unexpected cookie value: %s\n", c.Value)
				}
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{}`))
			})
		},
		auther: SetCookie(&http.Cookie{Name: "cow", Value: "moo"}),
		setup: func(t *testing.T, c *client) {
			c.Client.Client.Transport = http.DefaultTransport
		},
		authStatus: kivik.StatusBadAPICall,
		authErr:    "kivik: HTTP client transport already set",
	})

	driver := &Couch{}
	tests.Run(t, func(t *testing.T, test tst) {
		s := httptest.NewServer(test.handler(t))
		defer s.Close()
		driverClient, err := driver.NewClient(s.URL)
		if err != nil {
			t.Fatal(err)
		}
		client := driverClient.(*client)
		if test.setup != nil {
			test.setup(t, client)
		}
		err = client.Authenticate(context.Background(), test.auther)
		testy.StatusError(t, test.authErr, test.authStatus, err)
		_, err = client.Version(context.Background())
		testy.StatusErrorRE(t, test.err, test.status, err)
	})
}
