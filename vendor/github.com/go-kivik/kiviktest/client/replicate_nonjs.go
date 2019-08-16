// +build !js

package client

import (
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func replicationOptions(_ *kt.Context, _ *kivik.Client, _, _, _ string, in map[string]interface{}) map[string]interface{} {
	if in == nil {
		in = make(map[string]interface{})
	}
	return in
}
