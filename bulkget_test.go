package couchdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"unicode"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
)

func TestBulkGet(t *testing.T) {
	type tst struct {
		db      *db
		docs    []driver.BulkGetReference
		options map[string]interface{}
		status  int
		err     string

		rowStatus int
		rowErr    string

		expected *driver.Row
	}
	tests := testy.NewTable()
	tests.Add("network error", tst{
		db: &db{
			client: newTestClient(nil, errors.New("random network error")),
		},
		status: kivik.StatusNetworkError,
		err:    "Post http://example.com/_bulk_get: random network error",
	})
	tests.Add("valid document", tst{
		db: &db{
			client: newTestClient(&http.Response{
				StatusCode: http.StatusOK,
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: ioutil.NopCloser(strings.NewReader(removeSpaces(`{
	  "results": [
	    {
	      "id": "foo",
	      "docs": [
	        {
	          "ok": {
	            "_id": "foo",
	            "_rev": "4-753875d51501a6b1883a9d62b4d33f91",
	            "value": "this is foo"
	          }
	        }
	      ]
	    }
	]`))),
			}, nil),
			dbName: "xxx",
		},
		expected: &driver.Row{
			ID:  "foo",
			Doc: []byte(`{"_id":"foo","_rev":"4-753875d51501a6b1883a9d62b4d33f91","value":"thisisfoo"}`),
		},
	})
	tests.Add("invalid id", tst{
		db: &db{
			client: newTestClient(&http.Response{
				StatusCode: http.StatusOK,
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       ioutil.NopCloser(strings.NewReader(`{"results": [{"id": "", "docs": [{"error":{"id":"","rev":null,"error":"illegal_docid","reason":"Document id must not be empty"}}]}]}`)),
			}, nil),
			dbName: "xxx",
		},
		docs: []driver.BulkGetReference{{ID: ""}},
		expected: &driver.Row{
			Error: &BulkGetError{
				ID:     "",
				Rev:    "",
				Err:    "illegal_docid",
				Reason: "Document id must not be empty",
			},
		},
	})
	tests.Add("not found", tst{
		db: &db{
			client: newTestClient(&http.Response{
				StatusCode: http.StatusOK,
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       ioutil.NopCloser(strings.NewReader(`{"results": [{"id": "asdf", "docs": [{"error":{"id":"asdf","rev":"1-xxx","error":"not_found","reason":"missing"}}]}]}`)),
			}, nil),
			dbName: "xxx",
		},
		docs: []driver.BulkGetReference{{ID: ""}},
		expected: &driver.Row{
			ID: "asdf",
			Error: &BulkGetError{
				ID:     "asdf",
				Rev:    "1-xxx",
				Err:    "not_found",
				Reason: "missing",
			},
		},
	})
	tests.Add("revs", tst{
		db: &db{
			client: newCustomClient(func(r *http.Request) (*http.Response, error) {
				revs := r.URL.Query().Get("revs")
				if revs != "true" {
					return nil, errors.New("Expected revs=true")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					ProtoMajor: 1,
					ProtoMinor: 1,
					Body:       ioutil.NopCloser(strings.NewReader(`{"results": [{"id": "test1", "docs": [{"ok":{"_id":"test1","_rev":"4-8158177eb5931358b3ddaadd6377cf00","moo":123,"oink":true,"_revisions":{"start":4,"ids":["8158177eb5931358b3ddaadd6377cf00","1c08032eef899e52f35cbd1cd5f93826","e22bea278e8c9e00f3197cb2edee8bf4","7d6ff0b102072755321aa0abb630865a"]},"_attachments":{"foo.txt":{"content_type":"text/plain","revpos":2,"digest":"md5-WiGw80mG3uQuqTKfUnIZsg==","length":9,"stub":true}}}}]}]}`)),
				}, nil
			}),
			dbName: "xxx",
		},
		options: map[string]interface{}{
			"revs": true,
		},
		expected: &driver.Row{
			ID:  "test1",
			Doc: []byte(`{"_id":"test1","_rev":"4-8158177eb5931358b3ddaadd6377cf00","moo":123,"oink":true,"_revisions":{"start":4,"ids":["8158177eb5931358b3ddaadd6377cf00","1c08032eef899e52f35cbd1cd5f93826","e22bea278e8c9e00f3197cb2edee8bf4","7d6ff0b102072755321aa0abb630865a"]},"_attachments":{"foo.txt":{"content_type":"text/plain","revpos":2,"digest":"md5-WiGw80mG3uQuqTKfUnIZsg==","length":9,"stub":true}}}`),
		},
	})

	tests.Run(t, func(t *testing.T, test tst) {
		rows, err := test.db.BulkGet(context.Background(), test.docs, test.options)
		testy.StatusError(t, test.err, test.status, err)

		row := new(driver.Row)
		err = rows.Next(row)
		defer rows.Close() // nolint: errcheck
		testy.StatusError(t, test.rowErr, test.rowStatus, err)

		if d := diff.Interface(test.expected, row); d != nil {
			t.Error(d)
		}
	})
}

var bulkGetInput = `
{
  "results": [
    {
      "id": "foo",
      "docs": [
        {
          "ok": {
            "_id": "foo",
            "_rev": "4-753875d51501a6b1883a9d62b4d33f91",
            "value": "this is foo",
            "_revisions": {
              "start": 4,
              "ids": [
                "753875d51501a6b1883a9d62b4d33f91",
                "efc54218773c6acd910e2e97fea2a608",
                "2ee767305024673cfb3f5af037cd2729",
                "4a7e4ae49c4366eaed8edeaea8f784ad"
              ]
            }
          }
        }
      ]
    },
    {
      "id": "foo",
      "docs": [
        {
          "ok": {
            "_id": "foo",
            "_rev": "1-4a7e4ae49c4366eaed8edeaea8f784ad",
            "value": "this is the first revision of foo",
            "_revisions": {
              "start": 1,
              "ids": [
                "4a7e4ae49c4366eaed8edeaea8f784ad"
              ]
            }
          }
        }
      ]
    },
    {
      "id": "bar",
      "docs": [
        {
          "ok": {
            "_id": "bar",
            "_rev": "2-9b71d36dfdd9b4815388eb91cc8fb61d",
            "baz": true,
            "_revisions": {
              "start": 2,
              "ids": [
                "9b71d36dfdd9b4815388eb91cc8fb61d",
                "309651b95df56d52658650fb64257b97"
              ]
            }
          }
        }
      ]
    },
    {
      "id": "baz",
      "docs": [
        {
          "error": {
            "id": "baz",
            "rev": "undefined",
            "error": "not_found",
            "reason": "missing"
          }
        }
      ]
    }
  ]
}
`

func TestGetBulkRowsIterator(t *testing.T) {
	type result struct {
		ID  string
		Err string
	}
	expected := []result{
		{ID: "foo"},
		{ID: "foo"},
		{ID: "bar"},
		{ID: "baz", Err: "not_found: missing"},
	}
	results := []result{}
	rows := newBulkGetRows(context.TODO(), ioutil.NopCloser(strings.NewReader(bulkGetInput)))
	var count int
	for {
		row := &driver.Row{}
		err := rows.Next(row)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next() failed: %s", err)
		}
		results = append(results, result{
			ID: row.ID,
			Err: func() string {
				if row.Error == nil {
					return ""
				}
				return row.Error.Error()
			}(),
		})
		if count++; count > 10 {
			t.Fatalf("Ran too many iterations.")
		}
	}
	if d := diff.Interface(expected, results); d != nil {
		t.Error(d)
	}
	if expected := 4; count != expected {
		t.Errorf("Expected %d rows, got %d", expected, count)
	}
	if err := rows.Next(&driver.Row{}); err != io.EOF {
		t.Errorf("Calling Next() after end returned unexpected error: %s", err)
	}
	if err := rows.Close(); err != nil {
		t.Errorf("Error closing rows iterator: %s", err)
	}
}

func removeSpaces(in string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, in)
}

func TestDecodeBulkResult(t *testing.T) {
	type tst struct {
		input    string
		err      string
		expected bulkResult
	}
	tests := testy.NewTable()
	tests.Add("real example", tst{
		input: removeSpaces(`{
      "id": "test1",
      "docs": [
        {
          "ok": {
            "_id": "test1",
            "_rev": "3-1c08032eef899e52f35cbd1cd5f93826",
            "moo": 123,
            "oink": false,
            "_attachments": {
              "foo.txt": {
                "content_type": "text/plain",
                "revpos": 2,
                "digest": "md5-WiGw80mG3uQuqTKfUnIZsg==",
                "length": 9,
                "stub": true
              }
            }
          }
        }
      ]
    }`),
		expected: bulkResult{
			ID: "test1",
			Docs: []bulkResultDoc{{
				Doc: json.RawMessage(`{"_id":"test1","_rev":"3-1c08032eef899e52f35cbd1cd5f93826","moo":123,"oink":false,"_attachments":{"foo.txt":{"content_type":"text/plain","revpos":2,"digest":"md5-WiGw80mG3uQuqTKfUnIZsg==","length":9,"stub":true}}}`),
			}},
		},
	})

	tests.Run(t, func(t *testing.T, test tst) {
		var result bulkResult
		err := json.Unmarshal([]byte(test.input), &result)
		testy.Error(t, test.err, err)
		if d := diff.Interface(test.expected, result); d != nil {
			t.Error(d)
		}
	})
}
