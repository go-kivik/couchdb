// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package chttp

import (
	"context"
	"net/http"
	"time"

	kivik "github.com/go-kivik/kivik/v4"
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
	transport  http.RoundTripper
	authExpiry *time.Time
}

var _ Authenticator = &CookieAuth{}

// Authenticate initiates a session with the CouchDB server.
func (a *CookieAuth) Authenticate(c *Client) error {
	a.client = c
	a.setCookieJar()
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
// It also drops the auth cookie if we receive a 401 response to ensure
// that follow up requests can try to authenticate again.
func (a *CookieAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := a.authenticate(req); err != nil {
		return nil, err
	}

	res, err := a.transport.RoundTrip(req)
	if err != nil {
		return res, err
	}

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		if cookie := a.Cookie(); cookie != nil {
			// set to expire yesterday to allow us to ditch it
			cookie.Expires = time.Now().AddDate(0, 0, -1)
			a.client.Jar.SetCookies(a.client.dsn, []*http.Cookie{cookie})
			a.client.authMU.Lock()
			a.authExpiry = nil
			a.client.authMU.Unlock()
		}
	}
	return res, nil
}

// shouldAuth returns true if there is no cookie set, or if it has expired.
func (a *CookieAuth) shouldAuth(req *http.Request) bool {
	if _, err := req.Cookie(kivik.SessionCookieName); err == nil {
		return false
	}
	if a.authExpiry == nil {
		return true
	}
	if !a.authExpiry.IsZero() {
		return a.authExpiry.Before(time.Now())
	}
	// If we get here, it means the server did not include an expiry time in
	// the session cookie. Some CouchDB configurations do this, but rather than
	// re-authenticating for every request, we'll let the session expire. A
	// future change might be to make a client-configurable option to set the
	// re-authentication timeout.
	return false
}

func (a *CookieAuth) authenticate(req *http.Request) error {
	ctx := req.Context()
	if inProg, _ := ctx.Value(authInProgress).(bool); inProg {
		return nil
	}
	a.client.authMU.Lock()
	defer a.client.authMU.Unlock()
	if !a.shouldAuth(req) {
		return nil
	}
	ctx = context.WithValue(ctx, authInProgress, true)
	opts := &Options{
		GetBody: BodyEncoder(a),
		Header: http.Header{
			HeaderIdempotencyKey: []string{},
		},
	}
	res, err := a.client.DoError(ctx, http.MethodPost, "/_session", opts)
	if err != nil {
		return err
	}
	for _, cookie := range res.Cookies() {
		if cookie.Name == kivik.SessionCookieName {
			expiry := cookie.Expires
			if !expiry.IsZero() {
				expiry = expiry.Add(-time.Minute)
			}
			a.authExpiry = &expiry
			break
		}
	}

	cookies := req.Cookies()
	req.Header.Del("Cookie")
	for _, cookie := range cookies {
		if cookie.Name != kivik.SessionCookieName {
			req.AddCookie(cookie)
		}
	}
	if c := a.Cookie(); c != nil {
		req.AddCookie(c)
	}
	return nil
}
