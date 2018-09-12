package couchdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
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
			status: kivik.StatusBadAPICall,
			err:    `parse http://foo.com/%xxx: invalid URL escape "%xx"`,
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
			result, err := driver.NewClient(test.dsn)
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := result.(*client); !ok {
				t.Errorf("Unexpected type returned: %t", result)
			}
		})
	}
}

func TestDB(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		dbName   string
		options  map[string]interface{}
		expected *db
		status   int
		err      string
	}{
		{
			name:   "no dbname",
			status: kivik.StatusBadRequest,
			err:    "kivik: dbName required",
		},
		{
			name:   "no full commit",
			dbName: "foo",
			expected: &db{
				dbName: "foo",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.DB(context.Background(), test.dbName, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if _, ok := result.(*db); !ok {
				t.Errorf("Unexpected result type: %T", result)
			}
		})
	}
}
