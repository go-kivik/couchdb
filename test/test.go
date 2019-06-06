package test

// RegisterCouchDBSuites registers the CouchDB related integration test suites.
func RegisterCouchDBSuites() {
	registerSuiteCouch16()
	registerSuiteCouch17()
	registerSuiteCouch20()
	registerSuiteCouch21()
	registerSuiteCouch22()
	registerSuiteCouch23()
	registerSuiteCloudant()
}
