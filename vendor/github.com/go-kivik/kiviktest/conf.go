package kiviktest

import "github.com/go-kivik/kiviktest/kt"

var suites = make(map[string]kt.SuiteConfig)

// RegisterSuite registers a Suite as available for testing.
func RegisterSuite(suite string, conf kt.SuiteConfig) {
	if _, ok := suites[suite]; ok {
		panic(suite + " already registered")
	}
	suites[suite] = conf
}
