package db

import (
	"context"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("Put", put)
}

func put(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			testPut(ctx, ctx.Admin)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			testPut(ctx, ctx.NoAuth)
		})
	})
}

func testPut(ctx *kt.Context, client *kivik.Client) {
	ctx.Parallel()
	dbName := ctx.TestDB()
	defer ctx.DestroyDB(dbName)
	db := client.DB(context.Background(), dbName, ctx.Options("db"))
	if err := db.Err(); !ctx.IsExpectedSuccess(err) {
		return
	}
	ctx.Run("group", func(ctx *kt.Context) {
		ctx.Run("Create", func(ctx *kt.Context) {
			ctx.Parallel()

			doc := &testDoc{
				ID:   ctx.TestDBName(),
				Name: "Alberto",
				Age:  32,
			}
			var rev string
			err := kt.Retry(func() error {
				var e error
				rev, e = db.Put(context.Background(), doc.ID, doc)
				return e
			})
			if !ctx.IsExpectedSuccess(err) {
				return
			}
			doc.Rev = rev
			doc.Age = 40
			ctx.Run("Update", func(ctx *kt.Context) {
				err := kt.Retry(func() error {
					_, e := db.Put(context.Background(), doc.ID, doc)
					return e
				})
				ctx.CheckError(err)
			})
		})
		ctx.Run("DesignDoc", func(ctx *kt.Context) {
			ctx.Parallel()
			doc := map[string]interface{}{
				"_id":      "_design/testddoc",
				"language": "javascript",
				"views": map[string]interface{}{
					"testview": map[string]interface{}{
						"map": `function(doc) {
			                if (doc.include) {
			                    emit(doc._id, doc.index);
			                }
			            }`,
					},
				},
			}
			err := kt.Retry(func() error {
				_, err := db.Put(context.Background(), doc["_id"].(string), doc)
				return err
			})
			ctx.CheckError(err)
		})
		ctx.Run("Local", func(ctx *kt.Context) {
			ctx.Parallel()
			doc := map[string]interface{}{
				"_id":  "_local/foo",
				"name": "Bob",
			}
			err := kt.Retry(func() error {
				_, err := db.Put(context.Background(), doc["_id"].(string), doc)
				return err
			})
			ctx.CheckError(err)
		})
		ctx.Run("LeadingUnderscoreInID", func(ctx *kt.Context) {
			ctx.Parallel()
			doc := map[string]interface{}{
				"_id":  "_badid",
				"name": "Bob",
			}
			err := kt.Retry(func() error {
				_, err := db.Put(context.Background(), doc["_id"].(string), doc)
				return err
			})
			ctx.CheckError(err)
		})
		ctx.Run("HeavilyEscapedID", func(ctx *kt.Context) {
			ctx.Parallel()
			doc := map[string]interface{}{
				"_id":  "foo+bar & spáces ?!*,",
				"name": "Bob",
			}
			err := kt.Retry(func() error {
				_, err := db.Put(context.Background(), doc["_id"].(string), doc)
				return err
			})
			ctx.CheckError(err)
		})
		ctx.Run("SlashInID", func(ctx *kt.Context) {
			ctx.Parallel()
			doc := map[string]interface{}{
				"_id":  "foo/bar",
				"name": "Bob",
			}
			err := kt.Retry(func() error {
				_, err := db.Put(context.Background(), doc["_id"].(string), doc)
				return err
			})
			ctx.CheckError(err)
		})
		ctx.Run("Conflict", func(ctx *kt.Context) {
			ctx.Parallel()
			doc := map[string]interface{}{
				"_id":  "duplicate",
				"name": "Bob",
			}
			err := kt.Retry(func() error {
				_, err := db.Put(context.Background(), doc["_id"].(string), doc)
				return err
			})
			if err != nil {
				ctx.Fatalf("Failed to create document for duplicate test: %s", err)
			}
			err = kt.Retry(func() error {
				_, err = db.Put(context.Background(), doc["_id"].(string), doc)
				return err
			})
			ctx.CheckError(err)
		})
	})
}
