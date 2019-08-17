package db

import (
	"context"
	"io"
	"io/ioutil"
	"strings"

	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func init() {
	kt.Register("PutAttachment", putAttachment)
}

func putAttachment(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		dbname := ctx.TestDB()
		defer ctx.DestroyDB(dbname)
		ctx.Run("group", func(ctx *kt.Context) {
			ctx.RunAdmin(func(ctx *kt.Context) {
				ctx.Parallel()
				testPutAttachment(ctx, ctx.Admin, dbname)
			})
			ctx.RunNoAuth(func(ctx *kt.Context) {
				ctx.Parallel()
				testPutAttachment(ctx, ctx.NoAuth, dbname)
			})
		})
	})
}

func testPutAttachment(ctx *kt.Context, client *kivik.Client, dbname string) {
	db := client.DB(context.Background(), dbname, ctx.Options("db"))
	if err := db.Err(); err != nil {
		ctx.Fatalf("Failed to open db: %s", err)
	}
	adb := ctx.Admin.DB(context.Background(), dbname, ctx.Options("db"))
	if err := adb.Err(); err != nil {
		ctx.Fatalf("Failed to open admin db: %s", err)
	}
	ctx.Run("Update", func(ctx *kt.Context) {
		ctx.Parallel()
		var docID, rev string
		err := kt.Retry(func() error {
			var e error
			docID, rev, e = adb.CreateDoc(context.Background(), map[string]string{"name": "Robert"})
			return e
		})
		if err != nil {
			ctx.Fatalf("Failed to create doc: %s", err)
		}
		err = kt.Retry(func() error {
			att := &kivik.Attachment{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     stringReadCloser("test content"),
			}
			_, err = db.PutAttachment(context.Background(), docID, rev, att)
			return err
		})
		ctx.CheckError(err)
	})
	ctx.Run("Create", func(ctx *kt.Context) {
		ctx.Parallel()
		docID := ctx.TestDBName()
		err := kt.Retry(func() error {
			att := &kivik.Attachment{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     stringReadCloser("test content"),
			}
			_, err := db.PutAttachment(context.Background(), docID, "", att)
			return err
		})
		ctx.CheckError(err)
	})
	ctx.Run("Conflict", func(ctx *kt.Context) {
		ctx.Parallel()
		var docID string
		err2 := kt.Retry(func() error {
			var e error
			docID, _, e = adb.CreateDoc(context.Background(), map[string]string{"name": "Robert"})
			return e
		})
		if err2 != nil {
			ctx.Fatalf("Failed to create doc: %s", err2)
		}
		err := kt.Retry(func() error {
			att := &kivik.Attachment{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     stringReadCloser("test content"),
			}
			_, err := db.PutAttachment(context.Background(), docID, "5-20bd3c7d7d6b81390c6679d8bae8795b", att)
			return err
		})
		ctx.CheckError(err)
	})
	ctx.Run("UpdateDesignDoc", func(ctx *kt.Context) {
		ctx.Parallel()
		docID := "_design/" + ctx.TestDBName()
		doc := map[string]string{
			"_id": docID,
		}
		var rev string
		err := kt.Retry(func() error {
			var err error
			rev, err = adb.Put(context.Background(), docID, doc)
			return err
		})
		if err != nil {
			ctx.Fatalf("Failed to create design doc: %s", err)
		}
		err = kt.Retry(func() error {
			att := &kivik.Attachment{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     stringReadCloser("test content"),
			}
			_, err = db.PutAttachment(context.Background(), docID, rev, att)
			return err
		})
		ctx.CheckError(err)
	})
	ctx.Run("CreateDesignDoc", func(ctx *kt.Context) {
		ctx.Parallel()
		docID := "_design/" + ctx.TestDBName()
		err := kt.Retry(func() error {
			att := &kivik.Attachment{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     stringReadCloser("test content"),
			}
			_, err := db.PutAttachment(context.Background(), docID, "", att)
			return err
		})
		ctx.CheckError(err)
	})
}

func stringReadCloser(str string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(str))
}
