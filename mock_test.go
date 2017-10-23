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
	return &db{
		dbName: "testdb",
		client: newTestClient(response, err),
	}
}

func newCustomDB(fn func(*http.Request) (*http.Response, error)) *db {
	return &db{
		dbName: "testdb",
		client: newCustomClient(fn),
	}
}

func newTestClient(response *http.Response, err error) *client {
	return newCustomClient(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			defer req.Body.Close() // nolint: errcheck
			if _, e := ioutil.ReadAll(req.Body); e != nil {
				return nil, e
			}
		}
		if err != nil {
			return nil, err
		}
		response := response
		response.Request = req
		return response, nil
	})
}

func newCustomClient(fn func(*http.Request) (*http.Response, error)) *client {
	chttpClient, _ := chttp.New(context.Background(), "http://example.com/")
	chttpClient.Client.Transport = customTransport(fn)
	return &client{
		Client: chttpClient,
	}
}
