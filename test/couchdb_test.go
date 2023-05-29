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

//go:build !js
// +build !js

package test

import (
	"testing"

	_ "github.com/go-kivik/couchdb/v4"
	kiviktest "github.com/go-kivik/kiviktest/v4"
)

func init() {
	RegisterCouchDBSuites()
}

func TestCouch16(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch16, "KIVIK_TEST_DSN_COUCH16", t)
}

func TestCouch17(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch17, "KIVIK_TEST_DSN_COUCH17", t)
}

func TestCouch20(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch20, "KIVIK_TEST_DSN_COUCH20", t)
}

func TestCouch21(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch21, "KIVIK_TEST_DSN_COUCH21", t)
}

func TestCouch22(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch22, "KIVIK_TEST_DSN_COUCH22", t)
}

func TestCouch23(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch23, "KIVIK_TEST_DSN_COUCH23", t)
}

func TestCouch30(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch30, "KIVIK_TEST_DSN_COUCH30", t)
}

func TestCouch31(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch31, "KIVIK_TEST_DSN_COUCH31", t)
}

func TestCouch32(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch32, "KIVIK_TEST_DSN_COUCH32", t)
}

func TestCouch33(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch33, "KIVIK_TEST_DSN_COUCH33", t)
}

func TestCloudant(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCloudant, "KIVIK_TEST_DSN_CLOUDANT", t)
}
