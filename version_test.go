package couchdb

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestVersion2(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		expected *driver.Version
		err      string
	}{
		{
			name:   "client error",
			client: newTestClient(nil, errors.New("omg error")),
			err:    "Get http://example.com/: omg error",
		},
		{
			name: "invalid JSON response",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"couchdb":"Welcome","uuid":"a902efb0fac143c2b1f97160796a6347","version":"1.6.1","vendor":{"name":[]}}`)),
			}, nil),
			err: "json: cannot unmarshal array into Go struct field .name of type string",
		},
		{
			name: "error response",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusInternalServerError,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			err: "Internal Server Error",
		},
		{
			name: "CouchDB 1.6.1",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"couchdb":"Welcome","uuid":"a902efb0fac143c2b1f97160796a6347","version":"1.6.1","vendor":{"version":"1.6.1","name":"The Apache Software Foundation"}}`)),
			}, nil),
			expected: &driver.Version{
				Version:     "1.6.1",
				Vendor:      "The Apache Software Foundation",
				RawResponse: []byte(`{"couchdb":"Welcome","uuid":"a902efb0fac143c2b1f97160796a6347","version":"1.6.1","vendor":{"version":"1.6.1","name":"The Apache Software Foundation"}}`),
			},
		},
		{
			name: "CouchDB 2.0.0",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"couchdb":"Welcome","version":"2.0.0","vendor":{"name":"The Apache Software Foundation"}}`)),
			}, nil),
			expected: &driver.Version{
				Version:     "2.0.0",
				Vendor:      "The Apache Software Foundation",
				RawResponse: []byte(`{"couchdb":"Welcome","version":"2.0.0","vendor":{"name":"The Apache Software Foundation"}}`),
			},
		},
		{
			name: "CouchDB 2.1.0",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"couchdb":"Welcome","version":"2.1.0","features":["scheduler"],"vendor":{"name":"The Apache Software Foundation"}}`)),
			}, nil),
			expected: &driver.Version{
				Version:     "2.1.0",
				Vendor:      "The Apache Software Foundation",
				Features:    []string{"scheduler"},
				RawResponse: []byte(`{"couchdb":"Welcome","version":"2.1.0","features":["scheduler"],"vendor":{"name":"The Apache Software Foundation"}}`),
			},
		},
		{
			name: "Cloudant 2017-10-23",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"couchdb":"Welcome","version":"2.0.0","vendor":{"name":"IBM Cloudant","version":"6365","variant":"paas"},"features":["geo","scheduler"]}`)),
			}, nil),
			expected: &driver.Version{
				Version:     "2.0.0",
				Vendor:      "IBM Cloudant",
				Features:    []string{"geo", "scheduler"},
				RawResponse: []byte(`{"couchdb":"Welcome","version":"2.0.0","vendor":{"name":"IBM Cloudant","version":"6365","variant":"paas"},"features":["geo","scheduler"]}`),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.Version(context.Background())
			testy.Error(t, test.err, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}
