package chttp

import (
	"regexp"
	"testing"

	"github.com/flimzy/testy"
)

// curlStatusErrorRE is a modified version of testy.StatusError, which handles
// exit statuses as well.
func curlStatusErrorRE(t *testing.T, expected string, status, eStatus int, actual error) {
	var err string
	var actualStatus, actualExitStatus int
	if actual != nil {
		err = actual.Error()
		actualStatus = testy.StatusCode(actual)
		actualExitStatus = ExitStatus(actual)
	}
	match, e := regexp.MatchString(expected, err)
	if e != nil {
		t.Fatal(e)
	}
	if !match {
		t.Errorf("Unexpected error: %s (expected %s)", err, expected)
	}
	if status != actualStatus {
		t.Errorf("Unexpected status code: %d (expected %d)", actualStatus, status)
	}
	if eStatus != actualExitStatus {
		t.Errorf("Unexpected exit status: %d (expected %d)", actualExitStatus, eStatus)
	}
	if actual != nil {
		t.SkipNow()
	}
}
