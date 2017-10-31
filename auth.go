package couchdb

import (
	"context"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/errors"
	"github.com/go-kivik/couchdb/chttp"
)

func (c *client) Authenticate(ctx context.Context, a interface{}) error {
	if auth, ok := a.(chttp.Authenticator); ok {
		return auth.Authenticate(ctx, c.Client)
	}
	return errors.Status(kivik.StatusUnknownError, "kivik: invalid authenticator")
}
