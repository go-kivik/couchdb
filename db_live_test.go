package couchdb_test

import (
	"context"
	"os"
	"testing"

	"gitlab.com/flimzy/testy"

	kivik "github.com/go-kivik/kivik/v3"
)

func TestQueries_1_x(t *testing.T) {
	dsn := os.Getenv("KIVIK_TEST_DSN_COUCH17")
	if dsn == "" {
		t.Skip("KIVIK_TEST_DSN_COUCH17 not configured")
	}

	client, err := kivik.New("couch", dsn)
	if err != nil {
		t.Fatal(err)
	}

	db := client.DB(context.Background(), "_users")
	rows, err := db.AllDocs(context.Background(), map[string]interface{}{
		"queries": []map[string]interface{}{
			{},
			{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close() // nolint:errcheck
	result := make([]interface{}, 0)
	for rows.Next() {
		if rows.EOQ() {
			result = append(result, map[string]interface{}{
				"EOQ":        true,
				"total_rows": rows.TotalRows(),
			})
			continue
		}
		result = append(result, map[string]interface{}{
			"_id": rows.ID(),
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if d := testy.DiffInterface(testy.Snapshot(t), result); d != nil {
		t.Error(d)
	}
}

func TestQueries_2_x(t *testing.T) {
	dsn := os.Getenv("KIVIK_TEST_DSN_COUCH23")
	if dsn == "" {
		dsn = os.Getenv("KIVIK_TEST_DSN_COUCH22")
	}
	if dsn == "" {
		t.Skip("Neither KIVIK_TEST_DSN_COUCH22 nor KIVIK_TEST_DSN_COUCH23 configured")
	}

	client, err := kivik.New("couch", dsn)
	if err != nil {
		t.Fatal(err)
	}

	db := client.DB(context.Background(), "_users")
	rows, err := db.AllDocs(context.Background(), map[string]interface{}{
		"queries": []map[string]interface{}{
			{},
			{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close() // nolint:errcheck
	result := make([]interface{}, 0)
	for rows.Next() {
		if rows.EOQ() {
			result = append(result, map[string]interface{}{
				"EOQ":        true,
				"total_rows": rows.TotalRows(),
			})
			continue
		}
		result = append(result, map[string]interface{}{
			"_id": rows.ID(),
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if d := testy.DiffInterface(testy.Snapshot(t), result); d != nil {
		t.Error(d)
	}
}

func TestQueries_3_x(t *testing.T) {
	dsn := os.Getenv("KIVIK_TEST_DSN_COUCH30")
	if dsn == "" {
		t.Skip("KIVIK_TEST_DSN_COUCH30 not configured")
	}

	client, err := kivik.New("couch", dsn)
	if err != nil {
		t.Fatal(err)
	}

	db := client.DB(context.Background(), "_users")
	rows, err := db.AllDocs(context.Background(), map[string]interface{}{
		"queries": []map[string]interface{}{
			{},
			{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close() // nolint:errcheck
	result := make([]interface{}, 0)
	for rows.Next() {
		if rows.EOQ() {
			result = append(result, map[string]interface{}{
				"EOQ":        true,
				"total_rows": rows.TotalRows(),
			})
			continue
		}
		result = append(result, map[string]interface{}{
			"_id": rows.ID(),
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if d := testy.DiffInterface(testy.Snapshot(t), result); d != nil {
		t.Error(d)
	}
}

// https://github.com/go-kivik/kivik/issues/509
func Test_bug509(t *testing.T) {
	dsn := os.Getenv("KIVIK_TEST_DSN_COUCH23")
	if dsn == "" {
		t.Skip("KIVIK_TEST_DSN_COUCH23 not configured")
	}

	client, err := kivik.New("couch", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close(context.Background())               // nolint:errcheck
	defer client.DestroyDB(context.Background(), "bug509") // nolint:errcheck
	if err := client.CreateDB(context.Background(), "bug509"); err != nil {
		t.Fatal(err)
	}

	db := client.DB(context.Background(), "bug509")
	if _, err := db.Put(context.Background(), "x", map[string]string{
		"_id": "x",
	}); err != nil {
		t.Fatal(err)
	}
	row := db.Get(context.Background(), "x", kivik.Options{
		"revs_info": true,
	})

	var doc map[string]interface{}
	if err := row.ScanDoc(&doc); err != nil {
		t.Fatal(err)
	}
}
