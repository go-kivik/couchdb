package db

import (
	"context"
	"sort"

	"github.com/flimzy/diff"
	"github.com/pkg/errors"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("Find", find)
}

func find(ctx *kt.Context) {
	ctx.RunAdmin(func(ctx *kt.Context) {
		testFind(ctx, ctx.Admin)
	})
	ctx.RunNoAuth(func(ctx *kt.Context) {
		testFind(ctx, ctx.NoAuth)
	})
	ctx.RunRW(func(ctx *kt.Context) {
		testFindRW(ctx)
	})
}

func testFindRW(ctx *kt.Context) {
	if ctx.Admin == nil {
		// Can't do anything here without admin access
		return
	}
	dbName, expected, err := setUpFindTest(ctx)
	if err != nil {
		ctx.Errorf("Failed to set up temp db: %s", err)
	}
	defer ctx.DestroyDB(dbName)
	ctx.Run("group", func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			doFindTest(ctx, ctx.Admin, dbName, 0, expected)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			doFindTest(ctx, ctx.NoAuth, dbName, 0, expected)
		})
	})
}

func setUpFindTest(ctx *kt.Context) (dbName string, docIDs []string, err error) {
	dbName = ctx.TestDB()
	db := ctx.Admin.DB(context.Background(), dbName, ctx.Options("db"))
	if err := db.Err(); err != nil {
		return dbName, nil, errors.Wrap(err, "failed to connect to db")
	}
	docIDs = make([]string, 10)
	for i := range docIDs {
		id := ctx.TestDBName()
		doc := struct {
			ID string `json:"id"`
		}{
			ID: id,
		}
		if _, err := db.Put(context.Background(), doc.ID, doc); err != nil {
			return dbName, nil, errors.Wrap(err, "failed to create doc")
		}
		docIDs[i] = id
	}
	sort.Strings(docIDs)
	return dbName, docIDs, nil

}

func testFind(ctx *kt.Context, client *kivik.Client) {
	if !ctx.IsSet("databases") {
		ctx.Errorf("databases not set; Did you configure this test?")
		return
	}
	for _, dbName := range ctx.StringSlice("databases") {
		func(dbName string) {
			ctx.Run(dbName, func(ctx *kt.Context) {
				doFindTest(ctx, client, dbName, int64(ctx.Int("offset")), ctx.StringSlice("expected"))
			})
		}(dbName)
	}
}

func doFindTest(ctx *kt.Context, client *kivik.Client, dbName string, expOffset int64, expected []string) {
	ctx.Parallel()
	db := client.DB(context.Background(), dbName, ctx.Options("db"))
	// Errors may be deferred here, so only return if we actually get
	// an error.
	if err := db.Err(); err != nil && !ctx.IsExpectedSuccess(err) {
		return
	}

	var rows *kivik.Rows
	err := kt.Retry(func() error {
		var e error
		rows, e = db.Find(context.Background(), `{"selector":{"_id":{"$gt":null}}}`)
		return e
	})

	if !ctx.IsExpectedSuccess(err) {
		return
	}
	docIDs := make([]string, 0, len(expected))
	for rows.Next() {
		var doc struct {
			DocID string `json:"_id"`
			Rev   string `json:"_rev"`
			ID    string `json:"id"`
		}
		if err := rows.ScanDoc(&doc); err != nil {
			ctx.Errorf("Failed to scan doc: %s", err)
		}
		docIDs = append(docIDs, doc.DocID)
	}
	if rows.Err() != nil {
		ctx.Fatalf("Failed to fetch row: %s", rows.Err())
	}
	sort.Strings(docIDs) // normalize order
	if d := diff.TextSlices(expected, docIDs); d != nil {
		ctx.Errorf("Unexpected document IDs returned:\n%s\n", d)
	}
	if rows.Offset() != expOffset {
		ctx.Errorf("Unexpected offset: %v", rows.Offset())
	}
	ctx.Run("Warning", func(ctx *kt.Context) {
		rows, err := db.Find(context.Background(), `{"selector":{"foo":{"$gt":null}}}`)
		if !ctx.IsExpectedSuccess(err) {
			return
		}
		for rows.Next() {
		}
		if w := ctx.String("warning"); w != rows.Warning() {
			ctx.Errorf("Warning:\nExpected: %s\n  Actual: %s", w, rows.Warning())
		}
	})
}
