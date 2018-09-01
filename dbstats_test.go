package couchdb

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
)

func TestStats(t *testing.T) {
	tests := []struct {
		name     string
		db       *db
		expected *driver.DBStats
		status   int
		err      string
	}{
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb: net error",
		},
		{
			name: "read error",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body: &mockReadCloser{
					ReadFunc: func(_ []byte) (int, error) {
						return 0, errors.New("read error")
					},
					CloseFunc: func() error { return nil },
				},
			}, nil),
			status: kivik.StatusNetworkError,
			err:    "read error",
		},
		{
			name: "invalid JSON response",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`invalid json`)),
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "error response",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "1.6.1",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":           {"Thu, 26 Oct 2017 12:58:14 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"235"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"db_name":"_users","doc_count":3,"doc_del_count":14,"update_seq":31,"purge_seq":0,"compact_running":false,"disk_size":127080,"data_size":6028,"instance_start_time":"1509022681259533","disk_format_version":6,"committed_update_seq":31}`)),
			}, nil),
			expected: &driver.DBStats{
				Name:         "_users",
				DocCount:     3,
				DeletedCount: 14,
				UpdateSeq:    "31",
				DiskSize:     127080,
				ActiveSize:   6028,
				RawResponse:  []byte(`{"db_name":"_users","doc_count":3,"doc_del_count":14,"update_seq":31,"purge_seq":0,"compact_running":false,"disk_size":127080,"data_size":6028,"instance_start_time":"1509022681259533","disk_format_version":6,"committed_update_seq":31}`),
			},
		},
		{
			name: "2.0.0",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: http.Header{
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"Date":                {"Thu, 26 Oct 2017 13:01:13 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"429"},
					"Cache-Control":       {"must-revalidate"},
					"X-Couch-Request-ID":  {"2486f27546"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"db_name":"_users","update_seq":"13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWQPVsOCS40DSE08WA0rLjUJIDX1eO3KYwGSDA1ACqhsPiF1CyDq9mclMuFVdwCi7j4hdQ8g6kDuywIAkRBjAw","sizes":{"file":87323,"external":2495,"active":6082},"purge_seq":0,"other":{"data_size":2495},"doc_del_count":6,"doc_count":1,"disk_size":87323,"disk_format_version":6,"data_size":6082,"compact_running":false,"instance_start_time":"0"}`)),
			}, nil),
			expected: &driver.DBStats{
				Name:         "_users",
				DocCount:     1,
				DeletedCount: 6,
				UpdateSeq:    "13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWQPVsOCS40DSE08WA0rLjUJIDX1eO3KYwGSDA1ACqhsPiF1CyDq9mclMuFVdwCi7j4hdQ8g6kDuywIAkRBjAw",
				DiskSize:     87323,
				ActiveSize:   6082,
				ExternalSize: 2495,
				RawResponse:  []byte(`{"db_name":"_users","update_seq":"13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWQPVsOCS40DSE08WA0rLjUJIDX1eO3KYwGSDA1ACqhsPiF1CyDq9mclMuFVdwCi7j4hdQ8g6kDuywIAkRBjAw","sizes":{"file":87323,"external":2495,"active":6082},"purge_seq":0,"other":{"data_size":2495},"doc_del_count":6,"doc_count":1,"disk_size":87323,"disk_format_version":6,"data_size":6082,"compact_running":false,"instance_start_time":"0"}`),
			},
		},
		{
			name: "2.1.1",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: http.Header{
					"Server":              {"CouchDB/2.0.0 (Erlang OTP/17)"},
					"Date":                {"Thu, 26 Oct 2017 13:01:13 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"429"},
					"Cache-Control":       {"must-revalidate"},
					"X-Couch-Request-ID":  {"2486f27546"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"db_name":"_users","update_seq":"13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWQPVsOCS40DSE08WA0rLjUJIDX1eO3KYwGSDA1ACqhsPiF1CyDq9mclMuFVdwCi7j4hdQ8g6kDuywIAkRBjAw","sizes":{"file":87323,"external":2495,"active":6082},"purge_seq":0,"other":{"data_size":2495},"doc_del_count":6,"doc_count":1,"disk_size":87323,"disk_format_version":6,"data_size":6082,"compact_running":false,"instance_start_time":"0","cluster":{"n":1,"q":2,"r":3,"w":4}}`)),
			}, nil),
			expected: &driver.DBStats{
				Name:         "_users",
				DocCount:     1,
				DeletedCount: 6,
				UpdateSeq:    "13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWQPVsOCS40DSE08WA0rLjUJIDX1eO3KYwGSDA1ACqhsPiF1CyDq9mclMuFVdwCi7j4hdQ8g6kDuywIAkRBjAw",
				DiskSize:     87323,
				ActiveSize:   6082,
				ExternalSize: 2495,
				Cluster: &driver.ClusterStats{
					Replicas:    1,
					Shards:      2,
					ReadQuorum:  3,
					WriteQuorum: 4,
				},
				RawResponse: []byte(`{"db_name":"_users","update_seq":"13-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWQPVsOCS40DSE08WA0rLjUJIDX1eO3KYwGSDA1ACqhsPiF1CyDq9mclMuFVdwCi7j4hdQ8g6kDuywIAkRBjAw","sizes":{"file":87323,"external":2495,"active":6082},"purge_seq":0,"other":{"data_size":2495},"doc_del_count":6,"doc_count":1,"disk_size":87323,"disk_format_version":6,"data_size":6082,"compact_running":false,"instance_start_time":"0","cluster":{"n":1,"q":2,"r":3,"w":4}}`),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.db.Stats(context.Background())
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDbsStats(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		dbnames  []string
		expected interface{}
		status   int
		err      string
	}{
		{
			name:    "network error",
			client:  newTestClient(nil, errors.New("net error")),
			dbnames: []string{"foo", "bar"},
			status:  kivik.StatusNetworkError,
			err:     "Post http://example.com/_dbs_info: net error",
		},
		{
			name: "read error",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body: &mockReadCloser{
					ReadFunc: func(_ []byte) (int, error) {
						return 0, errors.New("read error")
					},
					CloseFunc: func() error { return nil },
				},
			}, nil),
			status: kivik.StatusNetworkError,
			err:    "read error",
		},
		{
			name: "invalid JSON response",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`invalid json`)),
			}, nil),
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "error response",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "2.1.2",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusNotFound,
				Header: http.Header{
					"Server":              {"CouchDB/2.1.2 (Erlang OTP/17)"},
					"Date":                {"Sat, 01 Sep 2018 15:42:53 GMT"},
					"Content-Type":        {"application/json"},
					"Content-Length":      {"58"},
					"Cache-Control":       {"must-revalidate"},
					"X-Couch-Request-ID":  {"e1264663f9"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"error":"not_found","reason":"Database does not exist."}`)),
			}, nil),
			dbnames: []string{"foo", "bar"},
			err:     "Not Found",
			status:  kivik.StatusNotFound,
		},
		{
			name: "2.2.0",
			client: newTestClient(&http.Response{
				StatusCode: kivik.StatusOK,
				Header: http.Header{
					"Server":              {"CouchDB/2.2.0 (Erlang OTP/19)"},
					"Date":                {"Sat, 01 Sep 2018 15:50:56 GMT"},
					"Content-Type":        {"application/json"},
					"Transfer-Encoding":   {"chunked"},
					"Cache-Control":       {"must-revalidate"},
					"X-Couch-Request-ID":  {"1bf258cfbe"},
					"X-CouchDB-Body-Time": {"0"},
				},
				Body: ioutil.NopCloser(strings.NewReader(`[{"key":"foo","error":"not_found"},{"key":"bar","error":"not_found"},{"key":"_users","info":{"db_name":"_users","update_seq":"1-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWSPX40DSE08WA0jLjUJIDX1eM3JYwGSDA1ACqhsPiF1CyDq9hNSdwCi7j4hdQ8g6kDuywIAiVhi9w","sizes":{"file":24423,"external":5361,"active":2316},"purge_seq":0,"other":{"data_size":5361},"doc_del_count":0,"doc_count":1,"disk_size":24423,"disk_format_version":6,"data_size":2316,"compact_running":false,"cluster":{"q":8,"n":1,"w":1,"r":1},"instance_start_time":"0"}}]
`)),
			}, nil),
			expected: []*driver.DBStats{
				nil,
				nil,
				{
					Name:         "_users",
					DocCount:     1,
					UpdateSeq:    "1-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWSPX40DSE08WA0jLjUJIDX1eM3JYwGSDA1ACqhsPiF1CyDq9hNSdwCi7j4hdQ8g6kDuywIAiVhi9w",
					DiskSize:     24423,
					ActiveSize:   2316,
					ExternalSize: 5361,
					Cluster: &driver.ClusterStats{
						Replicas:    1,
						Shards:      8,
						ReadQuorum:  1,
						WriteQuorum: 1,
					},
					RawResponse: []byte(`{"db_name":"_users","update_seq":"1-g1AAAAEzeJzLYWBg4MhgTmHgzcvPy09JdcjLz8gvLskBCjMlMiTJ____PyuRAYeCJAUgmWSPX40DSE08WA0jLjUJIDX1eM3JYwGSDA1ACqhsPiF1CyDq9hNSdwCi7j4hdQ8g6kDuywIAiVhi9w","sizes":{"file":24423,"external":5361,"active":2316},"purge_seq":0,"other":{"data_size":5361},"doc_del_count":0,"doc_count":1,"disk_size":24423,"disk_format_version":6,"data_size":2316,"compact_running":false,"cluster":{"q":8,"n":1,"w":1,"r":1},"instance_start_time":"0"}`),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.DBsStats(context.Background(), test.dbnames)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}
