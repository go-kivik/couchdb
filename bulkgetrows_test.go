package couchdb

import (
	"encoding/json"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
)

func TestDecodeBulkResult(t *testing.T) {
	type tst struct {
		input    string
		err      string
		expected bulkResult
	}
	tests := testy.NewTable()
	tests.Add("real example", tst{
		input: `{
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
    }`,
		expected: bulkResult{
			Docs: []bulkResultDoc{{
				Doc: map[string]interface{}{
					"_id":  "test1",
					"_rev": "3-1c08032eef899e52f35cbd1cd5f93826",
					"moo":  float64(123),
					"oink": false,
					"_attachments": map[string]interface{}{
						"foo.txt": map[string]interface{}{
							"content_type": "text/plain",
							"revpos":       float64(2),
							"digest":       "md5-WiGw80mG3uQuqTKfUnIZsg==",
							"length":       float64(9),
							"stub":         true,
						},
					},
				},
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
