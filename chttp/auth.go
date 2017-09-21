package chttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"

	"github.com/flimzy/kivik"

	"golang.org/x/net/publicsuffix"
)

// Authenticator is an interface that provides authentication to a server.
type Authenticator interface {
	Authenticate(context.Context, *Client) error
	Logout(context.Context, *Client) error
}

// BasicAuth provides HTTP Basic Auth for a client.
type BasicAuth struct {
	Username string
	Password string

	// transport stores the original transport that is overridden by this auth
	// mechanism
	transport http.RoundTripper
}

var _ Authenticator = &BasicAuth{}

// RoundTrip fulfills the http.RoundTripper interface. It sets HTTP Basic Auth
// on outbound requests.
func (a *BasicAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(a.Username, a.Password)
	transport := a.transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(req)
}

// Authenticate sets HTTP Basic Auth headers for the client.
func (a *BasicAuth) Authenticate(ctx context.Context, c *Client) error {
	// First see if the credentials seem good
	req, err := c.NewRequest(ctx, kivik.MethodGet, "/_session", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.Username, a.Password)
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err = ResponseError(res); err != nil {
		return err
	}
	result := struct {
		Ctx struct {
			Name string `json:"name"`
		} `json:"userCtx"`
	}{}
	if err = json.NewDecoder(res.Body).Decode(&result); err != nil {
		return err
	}
	if result.Ctx.Name != a.Username {
		return errors.New("authentication failed")
	}
	// Everything looks good, lets make this official
	a.transport = c.Transport
	c.Transport = a
	return nil
}

// Logout unsets BasicAuthentication
func (a *BasicAuth) Logout(_ context.Context, c *Client) error {
	if c.Transport != a {
		return errors.New("Not registered as authenticator")
	}
	c.Transport = a.transport
	return nil
}

// CookieAuth provides CouchDB Cookie auth services as described at
// http://docs.couchdb.org/en/2.0.0/api/server/authn.html#cookie-authentication
type CookieAuth struct {
	Username string `json:"name"`
	Password string `json:"password"`

	// Cookie will be set to the CouchDB auth cookie returned by the server, once
	// authentication is successfully completed.
	Cookie *http.Cookie `json:"-"`

	transport http.RoundTripper
	// Set to true if the authenticator created the cookie jar; It will then
	// also destroy it on logout.
	setJar bool
}

var _ Authenticator = &CookieAuth{}

// Authenticate initiates a session with the CouchDB server.
func (a *CookieAuth) Authenticate(ctx context.Context, c *Client) error {
	if err := a.setCookieJar(c); err != nil {
		return err
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(a); err != nil {
		return err
	}
	if _, err := c.DoError(ctx, kivik.MethodPost, "/_session", &Options{Body: buf}); err != nil {
		return err
	}
	for _, cookie := range c.Jar.Cookies(c.dsn) {
		if cookie.Name == kivik.SessionCookieName {
			a.Cookie = cookie
			break
		}
	}
	return ValidateAuth(ctx, a.Username, c)
}

// ValidateAuth validates that the requested username is authenticated.
func ValidateAuth(ctx context.Context, username string, client *Client) error {
	// This does a final request to validate that auth was successful. Cookies
	// may be filtered by a proxy, or a misconfigured client, so this check is
	// necessary.
	result := struct {
		Ctx struct {
			Name string `json:"name"`
		} `json:"userCtx"`
	}{}
	if _, err := client.DoJSON(ctx, "GET", "/_session", nil, &result); err != nil {
		return err
	}
	if result.Ctx.Name != username {
		return errors.New("authentication failed")
	}
	return nil
}

func (a *CookieAuth) setCookieJar(c *Client) error {
	// If a jar is already set, just use it
	if c.Jar != nil {
		return nil
	}
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}
	c.Jar = jar
	a.setJar = true
	return nil
}

// Logout deletes the remote session.
func (a *CookieAuth) Logout(ctx context.Context, c *Client) error {
	_, err := c.DoError(ctx, kivik.MethodDelete, "/_session", nil)
	if a.setJar {
		c.Jar = nil
	}
	return err
}
