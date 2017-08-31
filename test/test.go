package test

// RegisterCouchDBSuites registers the CouchDB related integration test suites.
func RegisterCouchDBSuites() {
	registerSuiteCouch16()
	registerSuiteCouch20()
	registerSuiteCloudant()
}
