// +build js

package client

import (
	"github.com/gopherjs/gopherjs/js"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func replicationOptions(ctx *kt.Context, client *kivik.Client, target, source, repID string, in map[string]interface{}) map[string]interface{} {
	if in == nil {
		in = make(map[string]interface{})
	}
	if ctx.String("mode") != "pouchdb" {
		in["_id"] = repID
		return in
	}
	in["source"] = js.Global.Get("PouchDB").New(source)
	in["target"] = js.Global.Get("PouchDB").New(target)
	return in
}
