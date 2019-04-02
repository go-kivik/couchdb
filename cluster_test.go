package couchdb

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik"
)

const optionEnsureDBsExist = "ensure_dbs_exist"

func TestClusterStatus(t *testing.T) {
	type tst struct {
		client   *client
		options  map[string]interface{}
		expected string
		status   int
		err      string
	}
	tests := testy.NewTable()
	tests.Add("network error", tst{
		client: newTestClient(nil, errors.New("network error")),
		status: kivik.StatusNetworkError,
		err:    "Get http://example.com/_cluster_setup: network error",
	})
	tests.Add("finished", tst{
		client: newTestClient(&http.Response{
			StatusCode: http.StatusOK,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: ioutil.NopCloser(strings.NewReader(`{"state":"cluster_finished"}`)),
		}, nil),
		expected: "cluster_finished",
	})
	tests.Add("invalid option", tst{
		client: newCustomClient(func(r *http.Request) (*http.Response, error) {
			return nil, nil
		}),
		options: map[string]interface{}{
			optionEnsureDBsExist: 1.0,
		},
		status: kivik.StatusBadAPICall,
		err:    "kivik: invalid type float64 for options",
	})
	tests.Add("invalid param", tst{
		client: newCustomClient(func(r *http.Request) (*http.Response, error) {
			result := []string{}
			err := json.Unmarshal([]byte(r.URL.Query().Get(optionEnsureDBsExist)), &result)
			return nil, &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: err}
		}),
		options: map[string]interface{}{
			optionEnsureDBsExist: "foo,bar,baz",
		},
		status: kivik.StatusBadRequest,
		err:    "Get http://example.com/_cluster_setup?ensure_dbs_exist=foo%2Cbar%2Cbaz: invalid character 'o' in literal false (expecting 'a')",
	})
	tests.Add("ensure dbs", func(t *testing.T) interface{} {
		return tst{
			client: newCustomClient(func(r *http.Request) (*http.Response, error) {
				input := r.URL.Query().Get(optionEnsureDBsExist)
				expected := []string{"foo", "bar", "baz"}
				result := []string{}
				err := json.Unmarshal([]byte(input), &result)
				if err != nil {
					t.Fatalf("Failed to parse `%s`: %s\n", input, err)
				}
				if d := diff.Interface(expected, result); d != nil {
					t.Errorf("Unexpected db list:\n%s", d)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: ioutil.NopCloser(strings.NewReader(`{"state":"cluster_finished"}`)),
				}, nil
			}),
			options: map[string]interface{}{
				optionEnsureDBsExist: `["foo","bar","baz"]`,
			},
			expected: "cluster_finished",
		}
	})

	tests.Run(t, func(t *testing.T, test tst) {
		result, err := test.client.ClusterStatus(context.Background(), test.options)
		testy.StatusError(t, test.err, test.status, err)
		if result != test.expected {
			t.Errorf("Unexpected result:\nExpected: %s\n  Actual: %s\n", test.expected, result)
		}
	})
}

func TestClusterSetup(t *testing.T) {
	type tst struct {
		client *client
		action interface{}
		status int
		err    string
	}
	tests := testy.NewTable()
	tests.Add("network error", tst{
		client: newTestClient(nil, errors.New("network error")),
		status: kivik.StatusNetworkError,
		err:    "Post http://example.com/_cluster_setup: network error",
	})
	tests.Add("invalid action", tst{
		client: newTestClient(nil, nil),
		action: func() {},
		status: kivik.StatusBadAPICall,
		err:    "Post http://example.com/_cluster_setup: json: unsupported type: func()",
	})
	tests.Add("success", func(t *testing.T) interface{} {
		return tst{
			client: newCustomClient(func(r *http.Request) (*http.Response, error) {
				expected := map[string]interface{}{
					"action": "finish_cluster",
				}
				result := map[string]interface{}{}
				if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				if d := diff.Interface(expected, result); d != nil {
					t.Errorf("Unexpected request body:\n%s\n", d)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
				}, nil
			}),
			action: map[string]interface{}{
				"action": "finish_cluster",
			},
		}
	})
	tests.Add("already finished", tst{
		client: newTestClient(&http.Response{
			StatusCode:    http.StatusBadRequest,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: 63,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: ioutil.NopCloser(strings.NewReader(`{"error":"bad_request","reason":"Cluster is already finished"}`)),
		}, nil),
		action: map[string]interface{}{
			"action": "finish_cluster",
		},
		status: kivik.StatusBadRequest,
		err:    "Bad Request: Cluster is already finished",
	})

	tests.Run(t, func(t *testing.T, test tst) {
		err := test.client.ClusterSetup(context.Background(), test.action)
		testy.StatusError(t, test.err, test.status, err)
	})
}
