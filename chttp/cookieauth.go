package chttp

import (
	"context"
	"net/http"

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
}

var _ Authenticator = &CookieAuth{}

// Authenticate initiates a session with the CouchDB server.
func (a *CookieAuth) Authenticate(ctx context.Context, c *Client) error {
	if err := a.setCookieJar(c); err != nil {
		return err // impossible error
	}
	a.client = c
	opts := &Options{
		Body: EncodeBody(a),
	}
	if _, err := c.DoError(ctx, kivik.MethodPost, "/_session", opts); err != nil {
		return err
	}
	return ValidateAuth(ctx, a.Username, c)
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
