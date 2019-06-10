package chttp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"strings"
)

type ProxyAuth struct {
	Username string
	Secret   string
	Roles    []string
	Headers  map[string]string

	transport http.RoundTripper
}

var _ Authenticator = &ProxyAuth{}

func (a *ProxyAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	// Default header names
	tokenHeaderName := "X-Auth-CouchDB-Token"
	usernameHeaderName := "X-Auth-CouchDB-UserName"
	rolesHeaderName := "X-Auth-CouchDB-Roles"

	// Override the token header if specified
	if val, ok := a.Headers["token"]; ok {
		tokenHeaderName = val
	}

	// Override the username header if specified
	if val, ok := a.Headers["username"]; ok {
		usernameHeaderName = val
	}

	// Override the roles header if specified
	if val, ok := a.Headers["roles"]; ok {
		rolesHeaderName = val
	}

	// Convert roles slice to comma separated values
	rolesCsv := strings.Join(a.Roles[:], ",")

	// If the secret is an empty string, do not calculate the token
	if a.Secret != "" {
		// Generate auth token
		// https://docs.couchdb.org/en/stable/config/auth.html#couch_httpd_auth/x_auth_token
		h := hmac.New(sha1.New, []byte(a.Secret))
		_, err := h.Write([]byte(a.Username))
		if err != nil {
			return nil, err
		}
		token := hex.EncodeToString(h.Sum(nil))
		req.Header.Set(tokenHeaderName, token)
	}

	// Add headers to request
	req.Header.Set(usernameHeaderName, a.Username)
	req.Header.Set(rolesHeaderName, rolesCsv)

	return a.transport.RoundTrip(req)
}

func (a *ProxyAuth) Authenticate(c *Client) error {
	a.transport = c.Transport
	if a.transport == nil {
		a.transport = http.DefaultTransport
	}
	c.Transport = a
	return nil
}
