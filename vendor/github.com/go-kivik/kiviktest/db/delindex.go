package db

import (
	"context"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("DeleteIndex", delindex)
}

func delindex(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			ctx.Parallel()
			testDelIndex(ctx, ctx.Admin)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			ctx.Parallel()
			testDelIndex(ctx, ctx.NoAuth)
		})
	})
}

func testDelIndex(ctx *kt.Context, client *kivik.Client) {
	dbname := ctx.TestDB()
	defer ctx.Admin.DestroyDB(context.Background(), dbname, ctx.Options("db")) // nolint: errcheck
	dba := ctx.Admin.DB(context.Background(), dbname, ctx.Options("db"))
	if err := dba.Err(); err != nil {
		ctx.Fatalf("Failed to open db as admin: %s", err)
	}
	if err := dba.CreateIndex(context.Background(), "foo", "bar", `{"fields":["foo"]}`); err != nil {
		ctx.Fatalf("Failed to create index: %s", err)
	}
	db := client.DB(context.Background(), dbname, ctx.Options("db"))
	if err := db.Err(); err != nil {
		ctx.Fatalf("Failed to open db: %s", err)
	}
	ctx.Run("group", func(ctx *kt.Context) {
		ctx.Run("ValidIndex", func(ctx *kt.Context) {
			ctx.Parallel()
			ctx.CheckError(db.DeleteIndex(context.Background(), "foo", "bar"))
		})
		ctx.Run("NotFoundDdoc", func(ctx *kt.Context) {
			ctx.Parallel()
			ctx.CheckError(db.DeleteIndex(context.Background(), "notFound", "bar"))
		})
		ctx.Run("NotFoundName", func(ctx *kt.Context) {
			ctx.Parallel()
			ctx.CheckError(db.DeleteIndex(context.Background(), "foo", "notFound"))
		})
	})
}
