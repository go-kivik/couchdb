package chttp

import (
	"testing"

	"github.com/flimzy/testy"
)

// curlStatusError is a modified version of testy.StatusError, which handles
// exit statuses as well.
func curlStatusError(t *testing.T, expected string, status, eStatus int, actual error) {
	var err string
	var actualStatus, actualExitStatus int
	if actual != nil {
		err = actual.Error()
		actualStatus = testy.StatusCode(actual)
		actualExitStatus = ExitStatus(actual)
	}
	if expected != err {
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
