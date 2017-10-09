package couchdb

import (
	"context"
	"net/http"

	"github.com/go-kivik/couchdb/chttp"
)

type dummyTransport struct {
	response *http.Response
	err      error
}

var _ http.RoundTripper = &dummyTransport{}

func (t *dummyTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return t.response, t.err
}

func newTestDB(response *http.Response, err error) *db {
	chttpClient, _ := chttp.New(context.Background(), "http://example.com/")
	chttpClient.Client.Transport = &dummyTransport{response: response, err: err}
	return &db{
		dbName: "testdb",
		client: &client{
			Client: chttpClient,
		},
	}
}
