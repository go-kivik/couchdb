package client

import (
	"context"
	"sort"

	"github.com/flimzy/diff"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("AllDBs", allDBs)
}

func allDBs(ctx *kt.Context) {
	ctx.RunAdmin(func(ctx *kt.Context) {
		testAllDBs(ctx, ctx.Admin, ctx.StringSlice("expected"))
	})
	ctx.RunNoAuth(func(ctx *kt.Context) {
		testAllDBs(ctx, ctx.NoAuth, ctx.StringSlice("expected"))
	})
	if ctx.RW && ctx.Admin != nil {
		ctx.Run("RW", func(ctx *kt.Context) {
			testAllDBsRW(ctx)
		})
	}
}

func testAllDBsRW(ctx *kt.Context) {
	dbName := ctx.TestDB()
	defer ctx.DestroyDB(dbName)
	expected := append(ctx.StringSlice("expected"), dbName)
	ctx.Run("group", func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			testAllDBs(ctx, ctx.Admin, expected)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			testAllDBs(ctx, ctx.NoAuth, expected)
		})
	})
}

func testAllDBs(ctx *kt.Context, client *kivik.Client, expected []string) {
	ctx.Parallel()
	allDBs, err := client.AllDBs(context.Background())
	if !ctx.IsExpectedSuccess(err) {
		return
	}
	sort.Strings(expected)
	sort.Strings(allDBs)
	if d := diff.TextSlices(expected, allDBs); d != nil {
		ctx.Errorf("AllDBs() returned unexpected list:\n%s\n", d)
	}
}
