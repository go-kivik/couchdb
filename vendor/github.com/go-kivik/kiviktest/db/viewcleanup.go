package db

import (
	"context"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("ViewCleanup", viewCleanup)
}

func viewCleanup(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			ctx.Parallel()
			testViewCleanup(ctx, ctx.Admin)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			ctx.Parallel()
			testViewCleanup(ctx, ctx.NoAuth)
		})
	})
}

func testViewCleanup(ctx *kt.Context, client *kivik.Client) {
	dbname := ctx.TestDB()
	defer ctx.DestroyDB(dbname)
	db := client.DB(context.Background(), dbname, ctx.Options("db"))
	if err := db.Err(); err != nil {
		ctx.Fatalf("Failed to connect to db: %s", err)
	}
	ctx.CheckError(db.ViewCleanup(context.Background()))
}
