package chttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/go-kivik/kivik"
)

// CookieAuth provides CouchDB Cookie auth services as described at
// http://docs.couchdb.org/en/2.0.0/api/server/authn.html#cookie-authentication
//
// CookieAuth stores authentication state after use, so should not be re-used.
type CookieAuth struct {
	Username string `json:"name"`
	Password string `json:"password"`

	client *Client
	// transport stores the original transport that is overridden by this auth
	// mechanism
	transport http.RoundTripper
}

var _ Authenticator = &CookieAuth{}

// Authenticate initiates a session with the CouchDB server.
func (a *CookieAuth) Authenticate(c *Client) error {
	a.setCookieJar(c)
	a.client = c
	a.transport = c.Transport
	if a.transport == nil {
		a.transport = http.DefaultTransport
	}
	c.Transport = a
	return nil
}

// Cookie returns the current session cookie if found, or nil if not.
func (a *CookieAuth) Cookie() *http.Cookie {
	if a.client == nil {
		return nil
	}
	for _, cookie := range a.client.Jar.Cookies(a.client.dsn) {
		if cookie.Name == kivik.SessionCookieName {
			return cookie
		}
	}
	return nil
}

var authInProgress = &struct{ name string }{"in progress"}

// RoundTrip fulfills the http.RoundTripper interface. It sets
// (re-)authenticates when the cookie has expired or is not yet set.
func (a *CookieAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if inProg, _ := ctx.Value(authInProgress).(bool); !inProg {
		// this means we aren't in the process of authenticating already
		if cookie := a.Cookie(); cookie == nil {
			// No cookie set, so attempt auth
			ctx = context.WithValue(ctx, authInProgress, true)
			opts := &Options{
				Body: EncodeBody(a),
			}
			if _, err := a.client.DoError(ctx, kivik.MethodPost, "/_session", opts); err != nil {
				return nil, err
			}
			// Now make sure the original request is authenticated
			req.AddCookie(a.Cookie())
		}
	}
	resp, err := a.transport.RoundTrip(req)
	if err != nil || resp.StatusCode >= 400 {
		x, _ := httputil.DumpRequest(req, false)
		fmt.Printf("%s\nERR: %s\n", (string(x)), err)
	}
	return resp, err
}
