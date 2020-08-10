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

package chttp

import (
	"net/url"
	"strings"
)

const (
	prefixDesign = "_design/"
	prefixLocal  = "_local/"
)

// EncodeDocID encodes a document ID according to CouchDB's path encoding rules.
//
// In particular:
// -  '_design/' and '_local/' prefixes are unaltered.
// - The rest of the docID is Query-URL encoded (despite being part of the path)
func EncodeDocID(docID string) string {
	for _, prefix := range []string{prefixDesign, prefixLocal} {
		if strings.HasPrefix(docID, prefix) {
			return prefix + url.QueryEscape(strings.TrimPrefix(docID, prefix))
		}
	}
	return url.QueryEscape(docID)
}
