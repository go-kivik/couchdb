package kt

import (
	"context"
)

// DestroyDB cleans up the specified DB after tests run
func (c *Context) DestroyDB(name string) {
	Retry(func() error { // nolint: errcheck
		return c.Admin.DestroyDB(context.Background(), name, c.Options("db"))
	})
}
