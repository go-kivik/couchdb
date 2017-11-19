package couchdb

// OptionFullCommit is the option key used to set the `X-Couch-Full-Commit`
// header in the request when set to true.
//
// Example:
//
//    db.Put(ctx, "doc_id", doc, kivik.Options{couchdb.ForceCommitOptionKey: true})
const OptionFullCommit = "X-Couch-Full-Commit"

// optionForceCommit is an unfortunately mispelled version of "full-commit",
// retained for backward compatibility.
const optionForceCommit = "force_commit"
