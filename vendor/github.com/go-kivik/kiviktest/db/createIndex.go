package db

import (
	"context"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("CreateIndex", createIndex)
}

func createIndex(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			ctx.Parallel()
			testCreateIndex(ctx, ctx.Admin)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			ctx.Parallel()
			testCreateIndex(ctx, ctx.NoAuth)
		})
	})
}

func testCreateIndex(ctx *kt.Context, client *kivik.Client) {
	dbname := ctx.TestDB()
	defer ctx.DestroyDB(dbname)
	db := client.DB(context.Background(), dbname, ctx.Options("db"))
	if err := db.Err(); err != nil {
		ctx.Fatalf("Failed to open db: %s", err)
	}
	ctx.Run("group", func(ctx *kt.Context) {
		ctx.Run("Valid", func(ctx *kt.Context) {
			doCreateIndex(ctx, db, `{"fields":["foo"]}`)
		})
		ctx.Run("NilIndex", func(ctx *kt.Context) {
			doCreateIndex(ctx, db, nil)
		})
		ctx.Run("BlankIndex", func(ctx *kt.Context) {
			doCreateIndex(ctx, db, "")
		})
		ctx.Run("EmptyIndex", func(ctx *kt.Context) {
			doCreateIndex(ctx, db, "{}")
		})
		ctx.Run("InvalidIndex", func(ctx *kt.Context) {
			doCreateIndex(ctx, db, `{"oink":true}`)
		})
		ctx.Run("InvalidJSON", func(ctx *kt.Context) {
			doCreateIndex(ctx, db, `chicken`)
		})
	})
}

func doCreateIndex(ctx *kt.Context, db *kivik.DB, index interface{}) {
	ctx.Parallel()
	err := kt.Retry(func() error {
		return db.CreateIndex(context.Background(), "", "", index)
	})
	ctx.CheckError(err)
}
