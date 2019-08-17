package kivik

const (
	// KivikVersion is the version of the Kivik library.
	KivikVersion = "2.0.0-prerelease"
	// KivikVendor is the vendor string reported by this library.
	KivikVendor = "Kivik"
)

// SessionCookieName is the name of the CouchDB session cookie.
const SessionCookieName = "AuthSession"

// UserPrefix is the mandatory CouchDB user prefix.
// See http://docs.couchdb.org/en/2.0.0/intro/security.html#org-couchdb-user
const UserPrefix = "org.couchdb.user:"

// EndKeySuffix is a high Unicode character (0xfff0) useful for appending to an
// endkey argument, when doing a ranged search, as described here:
// http://couchdb.readthedocs.io/en/latest/ddocs/views/collation.html#string-ranges
//
// Example, to return all results with keys beginning with "foo":
//
//    rows, err := db.Query(context.TODO(), "ddoc", "view", map[string]interface{}{
//        "startkey": `"foo"`,                           // Quotes are necessary so the key is
//        "endkey":   `"foo` + kivik.EndKeySuffix + `"`, // a valid JSON object
//    })
const EndKeySuffix = string(0xfff0)

// HTTP methods supported by CouchDB. This is almost an exact copy of the
// methods in the standard http package, with the addition of MethodCopy, and
// a few methods left out which are not used by CouchDB.
const (
	MethodGet    = "GET"
	MethodHead   = "HEAD"
	MethodPost   = "POST"
	MethodPut    = "PUT"
	MethodDelete = "DELETE"
	MethodCopy   = "COPY"
)

// HTTP response codes permitted by the CouchDB API.
// See http://docs.couchdb.org/en/2.1.2/api/basics.html#http-status-codes
const (
	StatusOK                           = 200
	StatusCreated                      = 201
	StatusAccepted                     = 202
	StatusFound                        = 302
	StatusNotModified                  = 304
	StatusBadRequest                   = 400
	StatusUnauthorized                 = 401
	StatusForbidden                    = 403
	StatusNotFound                     = 404
	StatusMethodNotAllowed             = 405
	StatusRequestTimeout               = 408
	StatusConflict                     = 409
	StatusPreconditionFailed           = 412
	StatusRequestEntityTooLarge        = 413
	StatusUnsupportedMediaType         = 415
	StatusRequestedRangeNotSatisfiable = 416
	StatusExpectationFailed            = 417
	StatusInternalServerError          = 500

	// StatusNotImplemented is not returned by CouchDB proper. It is used by
	// Kivik for optional features which are not implemented by some drivers.
	StatusNotImplemented = 501

	// StatusBadGateway is returned by Kivik in case of an error outside of the
	// HTTP transport. This could be the result of a network error, or having
	// received a malformed response from the server
	StatusBadGateway = 502

	// Error status over 600 are obviously not proper HTTP errors at all. They
	// are used for kivik-generated errors of various types.

	// StatusUnknownError is deprecated.
	StatusUnknownError = 500

	// StatusNetworkError is deprecated.
	StatusNetworkError = 502

	// StatusBadResponse is deprecated.
	StatusBadResponse = 502

	// StatusIteratorUnusable is deprecated
	StatusIteratorUnusable = 400

	// BadAPICall is deprecated.
	StatusBadAPICall = 400
)
