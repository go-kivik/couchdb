package couchdb

// Version is the current version of this package.
const Version = "2.0.0-prerelease"

const (
	// OptionFullCommit is the option key used to set the `X-Couch-Full-Commit`
	// header in the request when set to true.
	//
	// Example:
	//
	//    db.Put(ctx, "doc_id", doc, kivik.Options{couchdb.OptionFullCommit: true})
	OptionFullCommit = "X-Couch-Full-Commit"

	// OptionIfNoneMatch is an option key to set the If-None-Match header on
	// the request.
	//
	// Example:
	//
	//    row, err := db.Get(ctx, "doc_id", kivik.Options{couchdb.OptionIfNoneMatch: "1-xxx"})
	OptionIfNoneMatch = "If-None-Match"
)
