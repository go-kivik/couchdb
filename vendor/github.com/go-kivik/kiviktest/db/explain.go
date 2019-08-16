package db

import (
	"context"

	"github.com/flimzy/diff"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("Explain", explain)
}

func explain(ctx *kt.Context) {
	ctx.RunAdmin(func(ctx *kt.Context) {
		testExplain(ctx, ctx.Admin)
	})
	ctx.RunNoAuth(func(ctx *kt.Context) {
		testExplain(ctx, ctx.NoAuth)
	})
	ctx.RunRW(func(ctx *kt.Context) {
		testExplainRW(ctx)
	})
}

func testExplainRW(ctx *kt.Context) {
	if ctx.Admin == nil {
		// Can't do anything here without admin access
		return
	}
	dbName := ctx.TestDB()
	defer ctx.DestroyDB(dbName)
	ctx.Run("group", func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			doExplainTest(ctx, ctx.Admin, dbName)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			doExplainTest(ctx, ctx.NoAuth, dbName)
		})
	})
}

func testExplain(ctx *kt.Context, client *kivik.Client) {
	if !ctx.IsSet("databases") {
		ctx.Errorf("databases not set; Did you configure this test?")
		return
	}
	for _, dbName := range ctx.StringSlice("databases") {
		func(dbName string) {
			ctx.Run(dbName, func(ctx *kt.Context) {
				doExplainTest(ctx, client, dbName)
			})
		}(dbName)
	}
}

func doExplainTest(ctx *kt.Context, client *kivik.Client, dbName string) {
	ctx.Parallel()
	db := client.DB(context.Background(), dbName, ctx.Options("db"))
	// Errors may be deferred here, so only return if we actually get
	// an error.
	if err := db.Err(); err != nil && !ctx.IsExpectedSuccess(err) {
		return
	}

	var plan *kivik.QueryPlan
	err := kt.Retry(func() error {
		var e error
		plan, e = db.Explain(context.Background(), `{"selector":{"_id":{"$gt":null}}}`)
		return e
	})
	if !ctx.IsExpectedSuccess(err) {
		return
	}
	expected := new(kivik.QueryPlan)
	if e, ok := ctx.Interface("plan").(*kivik.QueryPlan); ok {
		*expected = *e // Make a shallow copy
	} else {
		expected = &kivik.QueryPlan{
			Index: map[string]interface{}{
				"ddoc": nil,
				"name": "_all_docs",
				"type": "special",
				"def":  map[string]interface{}{"fields": []interface{}{map[string]string{"_id": "asc"}}},
			},
			Selector: map[string]interface{}{"_id": map[string]interface{}{"$gt": nil}},
			Options: map[string]interface{}{
				"bookmark":  "nil",
				"conflicts": false,
				"fields":    "all_fields",
				"limit":     25,
				"r":         []int{49},
				"skip":      0,
				"sort":      map[string]interface{}{},
				"use_index": []interface{}{},
			},
			Limit: 25,
			Range: map[string]interface{}{
				"start_key": nil,
				"end_key":   "\xef\xbf\xbd",
			},
		}
	}
	expected.DBName = dbName
	if d := diff.AsJSON(expected, plan); d != nil {
		ctx.Errorf("Unexpected document IDs returned:\n%s\n", d)
	}
}
