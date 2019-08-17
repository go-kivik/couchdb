package db

import (
	"context"
	"fmt"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("Stats", stats)
}

func stats(ctx *kt.Context) {
	ctx.RunAdmin(func(ctx *kt.Context) {
		ctx.Parallel()
		roTests(ctx, ctx.Admin)
	})
	ctx.RunNoAuth(func(ctx *kt.Context) {
		ctx.Parallel()
		roTests(ctx, ctx.NoAuth)
	})
	ctx.RunRW(func(ctx *kt.Context) {
		ctx.Parallel()
		ctx.RunAdmin(func(ctx *kt.Context) {
			ctx.Parallel()
			rwTests(ctx, ctx.Admin)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			ctx.Parallel()
			rwTests(ctx, ctx.NoAuth)
		})
	})
}

func rwTests(ctx *kt.Context, client *kivik.Client) {
	dbname := ctx.TestDB()
	defer ctx.DestroyDB(dbname)
	db := ctx.Admin.DB(context.Background(), dbname, ctx.Options("db"))
	if err := db.Err(); err != nil {
		ctx.Fatalf("Failed to connect to db: %s", err)
	}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("%d", i)
		rev, err := db.Put(context.Background(), id, struct{}{})
		if err != nil {
			ctx.Fatalf("Failed to create document ID %s: %s", id, err)
		}
		if i > 5 {
			if _, err = db.Delete(context.Background(), id, rev); err != nil {
				ctx.Fatalf("Failed to delete document ID %s: %s", id, err)
			}
		}
	}
	testDBInfo(ctx, client, dbname, 6)
}

func roTests(ctx *kt.Context, client *kivik.Client) {
	for _, dbname := range ctx.MustStringSlice("databases") {
		func(dbname string) {
			ctx.Run(dbname, func(ctx *kt.Context) {
				ctx.Parallel()
				testDBInfo(ctx, client, dbname, 0)
			})
		}(dbname)
	}
}

func testDBInfo(ctx *kt.Context, client *kivik.Client, dbname string, docCount int64) {
	stats, err := client.DB(context.Background(), dbname, ctx.Options("db")).Stats(context.Background())
	if !ctx.IsExpectedSuccess(err) {
		return
	}
	if stats.Name != dbname {
		ctx.Errorf("Name: Expected '%s', actual '%s'", dbname, stats.Name)
	}
	if docCount > 0 {
		if docCount != stats.DocCount {
			ctx.Errorf("DocCount: Expected %d, actual %d", docCount, stats.DocCount)
		}
	}
}
