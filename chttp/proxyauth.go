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

	transport http.RoundTripper
}

var _ Authenticator = &ProxyAuth{}

func (a *ProxyAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	// Convert roles slice to comma separated values
	rolesCsv := strings.Join(a.Roles[:], ",")

	// Generate auth token
	// https://docs.couchdb.org/en/stable/config/auth.html#couch_httpd_auth/x_auth_token
	h := hmac.New(sha1.New, []byte(a.Secret))
	_, err := h.Write([]byte(a.Username))
	if err != nil {
		return nil, err
	}
	token := hex.EncodeToString(h.Sum(nil))

	// Add headers to request
	req.Header.Set("X-Auth-CouchDB-UserName", a.Username)
	req.Header.Set("X-Auth-CouchDB-Roles", rolesCsv)
	req.Header.Set("X-Auth-CouchDB-Token", token)

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
