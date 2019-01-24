/*
Package couchdb is a driver for connecting with a CouchDB server over HTTP.

Options

The CouchDB driver generally interprets kivik.Options keys and values as URL
query parameters. Values of the following types will be converted to their
appropriate string representation when URL-encoded:

 - bool
 - string
 - []string
 - int, uint, uint8, uint16, uint32, uint64, int8, int16, int32, int64

Passing any other type will return an error.

The only exceptions to the above rule are:

 - the special option keys defined by the package constants `OptionFullCommit`
   and `OptionIfNoneMatch`. These options set the appropriate HTTP request
   headers rather than setting a URL parameter.
 - the `keys` key, when passed to a view query, will result in a POST query
   being done, rather than a GET, to accomodate an arbitrary number of keys.

Authentication

The CouchDB driver supports a number of authentication methods. For most uses,
you don't need to worry about authentication at all--just include authentication
credentials in your connection DSN. This will use Cookie authentication by
default.

To use one of the explicit authentication mechanisms, you'll need to use kivik's
Authenticate method.  For example:

    client, _ := kivik.New("couch", "http://localhost:5984/")
    err := client.Authenticate(couchdb.BasicAuth("bob", "abc123"))

*/
package couchdb
