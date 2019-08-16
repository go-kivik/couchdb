package db

import (
	"context"

	"github.com/flimzy/diff"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("GetIndexes", getIndexes)
}

func getIndexes(ctx *kt.Context) {
	ctx.RunAdmin(func(ctx *kt.Context) {
		ctx.Parallel()
		roGetIndexesTests(ctx, ctx.Admin)
	})
	ctx.RunNoAuth(func(ctx *kt.Context) {
		ctx.Parallel()
		roGetIndexesTests(ctx, ctx.NoAuth)
	})
	ctx.RunRW(func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			ctx.Parallel()
			rwGetIndexesTests(ctx, ctx.Admin)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			ctx.Parallel()
			rwGetIndexesTests(ctx, ctx.NoAuth)
		})
	})
}

func roGetIndexesTests(ctx *kt.Context, client *kivik.Client) {
	databases := ctx.MustStringSlice("databases")
	for _, dbname := range databases {
		func(dbname string) {
			ctx.Run(dbname, func(ctx *kt.Context) {
				ctx.Parallel()
				testGetIndexes(ctx, client, dbname, ctx.Interface("indexes"))
			})
		}(dbname)
	}
}

func rwGetIndexesTests(ctx *kt.Context, client *kivik.Client) {
	dbname := ctx.TestDB()
	defer ctx.DestroyDB(dbname)
	dba := ctx.Admin.DB(context.Background(), dbname, ctx.Options("db"))
	if err := dba.Err(); err != nil {
		ctx.Fatalf("Failed to open db as admin: %s", err)
	}
	if err := dba.CreateIndex(context.Background(), "foo", "bar", `{"fields":["foo"]}`); err != nil {
		ctx.Fatalf("Failed to create index: %s", err)
	}
	indexes := ctx.Interface("indexes")
	if indexes == nil {
		indexes = []kivik.Index{
			kt.AllDocsIndex,
			{
				DesignDoc: "_design/foo",
				Name:      "bar",
				Type:      "json",
				Definition: map[string]interface{}{
					"fields": []map[string]string{
						{"foo": "asc"},
					},
				},
			},
		}
		testGetIndexes(ctx, client, dbname, indexes)
	}
}

func testGetIndexes(ctx *kt.Context, client *kivik.Client, dbname string, expected interface{}) {
	db := client.DB(context.Background(), dbname, ctx.Options("db"))
	if err := db.Err(); err != nil {
		ctx.Fatalf("Failed to open db: %s", err)
	}
	indexes, err := db.GetIndexes(context.Background())
	if !ctx.IsExpectedSuccess(err) {
		return
	}
	if d := diff.AsJSON(expected, indexes); d != nil {
		ctx.Errorf("Indexes differ from expectation:\n%s\n", d)
	}
}
