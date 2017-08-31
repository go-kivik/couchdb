package couchdb

import (
	"context"
	"errors"

	"github.com/go-kivik/couchdb/chttp"
)

func (c *client) Authenticate(ctx context.Context, a interface{}) error {
	if auth, ok := a.(chttp.Authenticator); ok {
		return auth.Authenticate(ctx, c.Client)
	}
	return errors.New("invalid authenticator")
}
