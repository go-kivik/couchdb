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
	Headers  http.Header

	transport http.RoundTripper
}

var _ Authenticator = &ProxyAuth{}

func (a *ProxyAuth) header(header string) string {
	if h := a.Headers.Get(header); h != "" {
		return http.CanonicalHeaderKey(h)
	}
	return header
}

func (a *ProxyAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	// Convert roles slice to comma separated values
	rolesCsv := strings.Join(a.Roles[:], ",")

	// If the secret is an empty string, do not calculate the token
	if a.Secret != "" {
		// Generate auth token
		// https://docs.couchdb.org/en/stable/config/auth.html#couch_httpd_auth/x_auth_token
		h := hmac.New(sha1.New, []byte(a.Secret))
		_, _ = h.Write([]byte(a.Username))
		token := hex.EncodeToString(h.Sum(nil))
		req.Header.Set(a.header("X-Auth-CouchDB-Token"), token)
	}

	// Add headers to request
	req.Header.Set(a.header("X-Auth-CouchDB-UserName"), a.Username)
	req.Header.Set(a.header("X-Auth-CouchDB-Roles"), rolesCsv)

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
