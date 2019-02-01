package couchdb

import (
	"context"
	"net/http"

	"github.com/go-kivik/couchdb/chttp"
)

const optionEnsureDBsExist = "ensure_dbs_exist"

func (c *client) ClusterStatus(ctx context.Context, opts map[string]interface{}) (string, error) {
	var result struct {
		State string `json:"state"`
	}
	query, err := optionsToParams(opts)
	if err != nil {
		return "", err
	}
	_, err = c.DoJSON(ctx, http.MethodGet, "/_cluster_setup", &chttp.Options{Query: query}, &result)
	return result.State, err
}
