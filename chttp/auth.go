package chttp

import (
	"context"
	"net/http/cookiejar"

	"golang.org/x/net/publicsuffix"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/errors"
)

// Authenticator is an interface that provides authentication to a server.
type Authenticator interface {
	Authenticate(context.Context, *Client) error
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
		return errors.Status(kivik.StatusBadResponse, "auth response for unexpected user")
	}
	return nil
}

func (a *CookieAuth) setCookieJar(c *Client) {
	// If a jar is already set, just use it
	if c.Jar != nil {
		return
	}
	// cookiejar.New never returns an error
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	c.Jar = jar
}
