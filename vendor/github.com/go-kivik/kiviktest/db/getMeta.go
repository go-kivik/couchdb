package db

import (
	"context"
	"strings"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("GetMeta", getMeta)
}

func getMeta(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		dbName := ctx.TestDB()
		defer ctx.DestroyDB(dbName)
		db := ctx.Admin.DB(context.Background(), dbName, ctx.Options("db"))
		if err := db.Err(); err != nil {
			ctx.Fatalf("Failed to connect to test db: %s", err)
		}
		doc := &testDoc{
			ID: "bob",
		}
		rev, err := db.Put(context.Background(), doc.ID, doc)
		if err != nil {
			ctx.Fatalf("Failed to create doc in test db: %s", err)
		}
		doc.Rev = rev

		ddoc := &testDoc{
			ID: "_design/foo",
		}
		rev, err = db.Put(context.Background(), ddoc.ID, ddoc)
		if err != nil {
			ctx.Fatalf("Failed to create design doc in test db: %s", err)
		}
		ddoc.Rev = rev

		local := &testDoc{
			ID: "_local/foo",
		}
		rev, err = db.Put(context.Background(), local.ID, local)
		if err != nil {
			ctx.Fatalf("Failed to create local doc in test db: %s", err)
		}
		local.Rev = rev

		ctx.Run("group", func(ctx *kt.Context) {
			ctx.RunAdmin(func(ctx *kt.Context) {
				ctx.Parallel()
				db := ctx.Admin.DB(context.Background(), dbName, ctx.Options("db"))
				if err := db.Err(); !ctx.IsExpectedSuccess(err) {
					return
				}
				testGetMeta(ctx, db, doc)
				testGetMeta(ctx, db, ddoc)
				testGetMeta(ctx, db, local)
				testGetMeta(ctx, db, &testDoc{ID: "bogus"})
			})
			ctx.RunNoAuth(func(ctx *kt.Context) {
				ctx.Parallel()
				db := ctx.NoAuth.DB(context.Background(), dbName, ctx.Options("db"))
				if err := db.Err(); !ctx.IsExpectedSuccess(err) {
					return
				}
				testGetMeta(ctx, db, doc)
				testGetMeta(ctx, db, ddoc)
				testGetMeta(ctx, db, local)
				testGetMeta(ctx, db, &testDoc{ID: "bogus"})
			})
		})
	})
}

func testGetMeta(ctx *kt.Context, db *kivik.DB, expectedDoc *testDoc) {
	ctx.Run(expectedDoc.ID, func(ctx *kt.Context) {
		ctx.Parallel()
		_, rev, err := db.GetMeta(context.Background(), expectedDoc.ID)
		if !ctx.IsExpectedSuccess(err) {
			return
		}
		doc := &testDoc{}
		if err = db.Get(context.Background(), expectedDoc.ID).ScanDoc(&doc); err != nil {
			ctx.Fatalf("Failed to scan doc: %s\n", err)
		}
		if strings.HasPrefix(expectedDoc.ID, "_local/") {
			// Revisions are meaningless for _local docs
			return
		}
		if rev != doc.Rev {
			ctx.Errorf("Unexpected rev. Expected: %s, Actual: %s", doc.Rev, rev)
		}
	})
}
