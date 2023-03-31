// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package test

import (
	"net/http"

	kiviktest "github.com/go-kivik/kiviktest/v4"
	"github.com/go-kivik/kiviktest/v4/kt"
)

// nolint:gomnd
func registerSuiteCouch17() {
	kiviktest.RegisterSuite(kiviktest.SuiteCouch17, kt.SuiteConfig{
		"AllDBs.expected": []string{"_replicator", "_users"},

		"CreateDB/RW/NoAuth.status":         http.StatusUnauthorized,
		"CreateDB/RW/Admin/Recreate.status": http.StatusPreconditionFailed,

		"DestroyDB/RW/NoAuth.status":              http.StatusUnauthorized,
		"DestroyDB/RW/Admin/NonExistantDB.status": http.StatusNotFound,

		"AllDocs.databases":                  []string{"_replicator", "chicken", "_duck"},
		"AllDocs/Admin/_replicator.expected": []string{"_design/_replicator"},
		"AllDocs/Admin/_replicator.offset":   0,
		"AllDocs/Admin/chicken.status":       http.StatusNotFound,
		"AllDocs/Admin/_duck.status":         http.StatusBadRequest,
		"AllDocs/NoAuth/_replicator.status":  http.StatusForbidden,
		"AllDocs/NoAuth/chicken.status":      http.StatusNotFound,
		"AllDocs/NoAuth/_duck.status":        http.StatusBadRequest,

		"Find.databases":     []string{"_users"},
		"Find.status":        http.StatusBadRequest, // Couchdb 1.7 doesn't support the find interface
		"CreateIndex.status": http.StatusBadRequest, // Couchdb 1.7 doesn't support the find interface
		"GetIndexes.skip":    true,                  // Couchdb 1.7 doesn't support the find interface
		"DeleteIndex.skip":   true,                  // Couchdb 1.7 doesn't support the find interface
		"Explain.databases":  []string{"_users"},
		"Explain.status":     http.StatusBadRequest, // Couchdb 1.7 doesn't support the find interface

		"DBExists.databases":              []string{"_users", "chicken", "_duck"},
		"DBExists/Admin/_users.exists":    true,
		"DBExists/Admin/chicken.exists":   false,
		"DBExists/Admin/_duck.status":     http.StatusBadRequest,
		"DBExists/NoAuth/_users.exists":   true,
		"DBExists/NoAuth/chicken.exists":  false,
		"DBExists/NoAuth/_duck.status":    http.StatusBadRequest,
		"DBExists/RW/group/Admin.exists":  true,
		"DBExists/RW/group/NoAuth.exists": true,

		"Log/NoAuth.status":                   http.StatusUnauthorized,
		"Log/NoAuth/Offset-1000.status":       http.StatusBadRequest,
		"Log/Admin/Offset-1000.status":        http.StatusBadRequest,
		"Log/Admin/HTTP/NegativeBytes.status": http.StatusInternalServerError,
		"Log/Admin/HTTP/TextBytes.status":     http.StatusInternalServerError,

		"Version.version":        `^1\.7\.2$`,
		"Version.vendor":         `^The Apache Software Foundation$`,
		"Version.vendor_version": `^1\.7\.2$`,

		"Get/RW/group/Admin/bogus.status":  http.StatusNotFound,
		"Get/RW/group/NoAuth/bogus.status": http.StatusNotFound,

		"GetRev/RW/group/Admin/bogus.status":  http.StatusNotFound,
		"GetRev/RW/group/NoAuth/bogus.status": http.StatusNotFound,

		"Flush.databases":                     []string{"_users", "chicken", "_duck"},
		"Flush/Admin/chicken/DoFlush.status":  http.StatusNotFound,
		"Flush/Admin/_duck/DoFlush.status":    http.StatusBadRequest,
		"Flush/NoAuth/chicken/DoFlush.status": http.StatusNotFound,
		"Flush/NoAuth/_duck/DoFlush.status":   http.StatusBadRequest,

		"Delete/RW/Admin/group/MissingDoc.status":        http.StatusNotFound,
		"Delete/RW/Admin/group/InvalidRevFormat.status":  http.StatusBadRequest,
		"Delete/RW/Admin/group/WrongRev.status":          http.StatusConflict,
		"Delete/RW/NoAuth/group/MissingDoc.status":       http.StatusNotFound,
		"Delete/RW/NoAuth/group/InvalidRevFormat.status": http.StatusBadRequest,
		"Delete/RW/NoAuth/group/WrongRev.status":         http.StatusConflict,
		"Delete/RW/NoAuth/group/DesignDoc.status":        http.StatusUnauthorized,

		"Session/Get/Admin.info.authentication_handlers":  "oauth,cookie,default",
		"Session/Get/Admin.info.authentication_db":        "_users",
		"Session/Get/Admin.info.authenticated":            "cookie",
		"Session/Get/Admin.userCtx.roles":                 "_admin",
		"Session/Get/Admin.ok":                            "true",
		"Session/Get/NoAuth.info.authentication_handlers": "oauth,cookie,default",
		"Session/Get/NoAuth.info.authentication_db":       "_users",
		"Session/Get/NoAuth.info.authenticated":           "",
		"Session/Get/NoAuth.userCtx.roles":                "",
		"Session/Get/NoAuth.ok":                           "true",

		"Session/Post/EmptyJSON.status":                             http.StatusBadRequest,
		"Session/Post/BogusTypeJSON.status":                         http.StatusUnauthorized,
		"Session/Post/BogusTypeForm.status":                         http.StatusUnauthorized,
		"Session/Post/EmptyForm.status":                             http.StatusUnauthorized,
		"Session/Post/BadJSON.status":                               http.StatusBadRequest,
		"Session/Post/BadForm.status":                               http.StatusUnauthorized,
		"Session/Post/MeaninglessJSON.status":                       http.StatusInternalServerError,
		"Session/Post/MeaninglessForm.status":                       http.StatusUnauthorized,
		"Session/Post/GoodJSON.status":                              http.StatusUnauthorized,
		"Session/Post/BadQueryParam.status":                         http.StatusUnauthorized,
		"Session/Post/BadCredsJSON.status":                          http.StatusUnauthorized,
		"Session/Post/BadCredsForm.status":                          http.StatusUnauthorized,
		"Session/Post/GoodCredsJSONRemoteRedirHeaderInjection.skip": true, // CouchDB allows header injection
		"Session/Post/GoodCredsJSONRemoteRedirInvalidURL.skip":      true, // CouchDB doesn't sanitize the Location value, so sends unparseable headers.
		"Session/Post/GoodCredsJSONRedirRelativeNoSlash.skip":       true, // As of Go 1.11.13 and 1.12.8, the result is rejected by Go for security reasons

		"Stats.databases":             []string{"_users", "chicken", "_duck"},
		"Stats/Admin/chicken.status":  http.StatusNotFound,
		"Stats/Admin/_duck.status":    http.StatusBadRequest,
		"Stats/NoAuth/chicken.status": http.StatusNotFound,
		"Stats/NoAuth/_duck.status":   http.StatusBadRequest,

		"Compact/RW/NoAuth.status": http.StatusUnauthorized,

		"ViewCleanup/RW/NoAuth.status": http.StatusUnauthorized,

		"Security.databases":              []string{"_replicator", "_users", "chicken", "_duck"},
		"Security/Admin/chicken.status":   http.StatusNotFound,
		"Security/Admin/_duck.status":     http.StatusBadRequest,
		"Security/NoAuth/chicken.status":  http.StatusNotFound,
		"Security/NoAuth/_duck.status":    http.StatusBadRequest,
		"Security/RW/group/NoAuth.status": http.StatusUnauthorized,

		"SetSecurity/RW/Admin/NotExists.status":  http.StatusNotFound,
		"SetSecurity/RW/NoAuth/NotExists.status": http.StatusNotFound,
		"SetSecurity/RW/NoAuth/Exists.status":    http.StatusUnauthorized,

		"DBUpdates/RW/NoAuth.status": http.StatusUnauthorized,

		"BulkDocs/RW/NoAuth/group/Mix/Conflict.status": http.StatusConflict,
		"BulkDocs/RW/Admin/group/Mix/Conflict.status":  http.StatusConflict,

		"GetAttachment/RW/group/Admin/foo/NotFound.status":  http.StatusNotFound,
		"GetAttachment/RW/group/NoAuth/foo/NotFound.status": http.StatusNotFound,

		"GetAttachmentMeta/RW/group/Admin/foo/NotFound.status":  http.StatusNotFound,
		"GetAttachmentMeta/RW/group/NoAuth/foo/NotFound.status": http.StatusNotFound,

		"PutAttachment/RW/group/Admin/Conflict.status":         http.StatusConflict,
		"PutAttachment/RW/group/NoAuth/Conflict.status":        http.StatusConflict,
		"PutAttachment/RW/group/NoAuth/UpdateDesignDoc.status": http.StatusUnauthorized,
		"PutAttachment/RW/group/NoAuth/CreateDesignDoc.status": http.StatusUnauthorized,

		// "DeleteAttachment/RW/group/Admin/NotFound.status":  http.StatusNotFound, // COUCHDB-3362
		// "DeleteAttachment/RW/group/NoAuth/NotFound.status": http.StatusNotFound, // COUCHDB-3362
		"DeleteAttachment/RW/group/Admin/NoDoc.status":      http.StatusConflict,
		"DeleteAttachment/RW/group/NoAuth/NoDoc.status":     http.StatusConflict,
		"DeleteAttachment/RW/group/NoAuth/DesignDoc.status": http.StatusUnauthorized,

		"Put/RW/Admin/group/LeadingUnderscoreInID.status":  http.StatusBadRequest,
		"Put/RW/Admin/group/Conflict.status":               http.StatusConflict,
		"Put/RW/NoAuth/group/LeadingUnderscoreInID.status": http.StatusBadRequest,
		"Put/RW/NoAuth/group/DesignDoc.status":             http.StatusUnauthorized,
		"Put/RW/NoAuth/group/Conflict.status":              http.StatusConflict,

		"GetReplications/NoAuth.status": http.StatusForbidden,

		"Replicate.NotFoundDB":                                  "http://localhost:5984/foo",
		"Replicate.timeoutSeconds":                              120,
		"Replicate.prefix":                                      "none",
		"Replicate/RW/NoAuth.status":                            http.StatusForbidden,
		"Replicate/RW/Admin/group/MissingSource/Results.status": http.StatusNotFound,
		"Replicate/RW/Admin/group/MissingTarget/Results.status": http.StatusNotFound,

		"Query/RW/group/Admin/WithoutDocs/ScanDoc.status":  http.StatusBadRequest,
		"Query/RW/group/NoAuth/WithoutDocs/ScanDoc.status": http.StatusBadRequest,

		"Changes/Continuous.options": map[string]interface{}{
			"feed":      "continuous",
			"since":     "now",
			"heartbeat": 6000,
		},
	})
}
