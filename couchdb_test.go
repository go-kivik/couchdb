package couchdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/testy"
)

func TestNewClient(t *testing.T) {
	type ncTest struct {
		name    string
		dsn     string
		status  int
		err     string
		cleanup func()
	}
	tests := []ncTest{
		{
			name:   "invalid url",
			dsn:    "foo.com/%xxx",
			status: kivik.StatusBadRequest,
			err:    `parse foo.com/%xxx: invalid URL escape "%xx"`,
		},
		func() ncTest {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			s := httptest.NewServer(handler)
			return ncTest{
				name: "success",
				dsn:  s.URL,
				cleanup: func() {
					s.Close()
				},
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.cleanup != nil {
				defer test.cleanup()
			}
			driver := &Couch{}
			result, err := driver.NewClient(context.Background(), test.dsn)
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := result.(*client); !ok {
				t.Errorf("Unexpected type returned: %t", result)
			}
		})
	}
}
