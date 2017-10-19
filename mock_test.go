package couchdb

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/go-kivik/couchdb/chttp"
)

type customTransport func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = customTransport(nil)

func (t customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t(req)
}

func newTestDB(response *http.Response, err error) *db {
	chttpClient, _ := chttp.New(context.Background(), "http://example.com/")
	chttpClient.Client.Transport = customTransport(func(req *http.Request) (*http.Response, error) {
		defer req.Body.Close()
		if _, e := ioutil.ReadAll(req.Body); e != nil {
			return nil, e
		}
		if err != nil {
			return nil, err
		}
		response := response
		response.Request = req
		return response, nil
	})
	return &db{
		dbName: "testdb",
		client: &client{
			Client: chttpClient,
		},
	}
}
