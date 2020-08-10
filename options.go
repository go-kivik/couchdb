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

package couchdb

import (
	"fmt"
	"net/http"

	kivik "github.com/go-kivik/kivik/v4"
)

func fullCommit(opts map[string]interface{}) (bool, error) {
	fc, ok := opts[OptionFullCommit]
	if !ok {
		return false, nil
	}
	fcBool, ok := fc.(bool)
	if !ok {
		return false, &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("kivik: option '%s' must be bool, not %T", OptionFullCommit, fc)}
	}
	delete(opts, OptionFullCommit)
	return fcBool, nil
}

func ifNoneMatch(opts map[string]interface{}) (string, error) {
	inm, ok := opts[OptionIfNoneMatch]
	if !ok {
		return "", nil
	}
	inmString, ok := inm.(string)
	if !ok {
		return "", &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("kivik: option '%s' must be string, not %T", OptionIfNoneMatch, inm)}
	}
	delete(opts, OptionIfNoneMatch)
	if inmString[0] != '"' {
		return `"` + inmString + `"`, nil
	}
	return inmString, nil
}
