package couchdb

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/go-kivik/couchdb/chttp"
)

type dummyTransport struct {
	response *http.Response
	err      error
}

var _ http.RoundTripper = &dummyTransport{}

func (t *dummyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	defer req.Body.Close()
	if _, err := ioutil.ReadAll(req.Body); err != nil {
		return nil, err
	}
	if t.err != nil {
		return nil, t.err
	}
	response := t.response
	response.Request = req
	return response, nil
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
