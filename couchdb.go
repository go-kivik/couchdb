// Package couchdb is a driver for connecting with a CouchDB server over HTTP.
package couchdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
)

// Couch represents the parent driver instance.
type Couch struct{}

var _ driver.Driver = &Couch{}

func init() {
	kivik.Register("couch", &Couch{})
}

// Known vendor strings
const (
	VendorCouchDB  = "The Apache Software Foundation"
	VendorCloudant = "IBM Cloudant"
)

type client struct {
	*chttp.Client

	// schedulerDetected will be set once the scheduler has been detected.
	// It should only be accessed through the schedulerSupported() method.
	schedulerDetected *bool
	sdMU              sync.Mutex
}

var _ driver.Client = &client{}

// NewClient establishes a new connection to a CouchDB server instance. If
// auth credentials are included in the URL, they are used to authenticate using
// CookieAuth (or BasicAuth if compiled with GopherJS). If you wish to use a
// different auth mechanism, do not specify credentials here, and instead call
// Authenticate() later.
func (d *Couch) NewClient(dsn string) (driver.Client, error) {
	chttpClient, err := chttp.New(dsn)
	if err != nil {
		return nil, err
	}
	chttpClient.UserAgents = []string{
		fmt.Sprintf("Kivik/%s", kivik.KivikVersion),
		fmt.Sprintf("Kivik CouchDB driver/%s", Version),
	}
	c := &client{
		Client: chttpClient,
	}
	return c, nil
}

func (c *client) DB(_ context.Context, dbName string, _ map[string]interface{}) (driver.DB, error) {
	if dbName == "" {
		return nil, missingArg("dbName")
	}
	return &db{
		client: c,
		dbName: dbName,
	}, nil
}
