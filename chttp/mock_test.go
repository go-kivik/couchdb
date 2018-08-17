package chttp

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type customTransport func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = customTransport(nil)

func (c customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return c(req)
}

func newCustomClient(dsn string, fn func(*http.Request) (*http.Response, error)) *Client {
	c := &Client{
		Client: &http.Client{
			Transport: customTransport(fn),
		},
	}
	var err error
	if dsn == "" {
		dsn = "http://example.com/"
	}
	c.dsn, err = url.Parse(dsn)
	if err != nil {
		panic(err)
	}
	return c
}

func newTestClient(resp *http.Response, err error) *Client {
	return newCustomClient("", func(_ *http.Request) (*http.Response, error) {
		return resp, err
	})
}

func Body(str string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(str))
}
