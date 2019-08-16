package testy

import (
	"net/http"
	"regexp"
	"testing"
)

// Error compares actual.Error() against expected, and triggers an error if
// they do not match. If actual is non-nil, t.SkipNow() is called as well.
func Error(t *testing.T, expected string, actual error) {
	helper(t)()
	var err string
	if actual != nil {
		err = actual.Error()
	}
	if expected != err {
		t.Errorf("Unexpected error: %s (expected %s)", err, expected)
	}
	if actual != nil {
		t.SkipNow()
	}
}

type statusCoder interface {
	StatusCode() int
}

// StatusCode returns the HTTP status code embedded in the error, or 500 if
// there is no specific status code.
func StatusCode(err error) int {
	if err == nil {
		return 0
	}
	if coder, ok := err.(statusCoder); ok {
		return coder.StatusCode()
	}
	return http.StatusInternalServerError
}

// StatusError compares actual.Error() and the embeded HTTP status code against
// expected, and triggers an error if they do not match. If actual is non-nil,
// t.SkipNow() is called as well.
func StatusError(t *testing.T, expected string, status int, actual error) {
	helper(t)()
	var err string
	var actualStatus int
	if actual != nil {
		err = actual.Error()
		actualStatus = StatusCode(actual)
	}
	if expected != err {
		t.Errorf("Unexpected error: %s (expected %s)", err, expected)
	}
	if status != actualStatus {
		t.Errorf("Unexpected status code: %d (expected %d)", actualStatus, status)
	}
	if actual != nil {
		t.SkipNow()
	}
}

// StatusErrorRE compares actual.Error() and the embeded HTTP status code against
// expected, and triggers an error if they do not match. If actual is non-nil,
// t.SkipNow() is called as well.
func StatusErrorRE(t *testing.T, expected string, status int, actual error) {
	helper(t)()
	var err string
	var actualStatus int
	if actual != nil {
		err = actual.Error()
		actualStatus = StatusCode(actual)
	}
	if (expected == "" && err != "") || (expected != "" && !regexp.MustCompile(expected).MatchString(err)) {
		t.Errorf("Unexpected error: %s (expected /%s/)", err, expected)
	}
	if status != actualStatus {
		t.Errorf("Unexpected status code: %d (expected %d)", actualStatus, status)
	}
	if actual != nil {
		t.SkipNow()
	}
}

// ErrorRE compares actual.Error() against expected, which is treated as a
// regular expression, and triggers an error if they do not match. If actual is
// non-nil, t.SkipNow() is called as well.
func ErrorRE(t *testing.T, expected string, actual error) {
	helper(t)()
	var err string
	if actual != nil {
		err = actual.Error()
	}
	if (expected == "" && err != "") || (expected != "" && !regexp.MustCompile(expected).MatchString(err)) {
		t.Errorf("Unexpected error: %s (expected /%s/)", err, expected)
	}
	if actual != nil {
		t.SkipNow()
	}
}

type exitStatuser interface {
	ExitStatus() int
}

// ExitStatus returns the exit status embedded in the error, or 1 (unknown
// error) if there is no specific status code.
func ExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if coder, ok := err.(exitStatuser); ok {
		return coder.ExitStatus()
	}
	return 1
}

// ExitStatusError compares actual.Error() and the embeded exit status against
// expected, and triggers an error if they do not match. If actual is non-nil,
// t.SkipNow() is called as well.
func ExitStatusError(t *testing.T, expected string, eStatus int, actual error) {
	helper(t)()
	var err string
	var actualEStatus int
	if actual != nil {
		err = actual.Error()
		actualEStatus = ExitStatus(actual)
	}
	if expected != err {
		t.Errorf("Unexpected error: %s (expected %s)", err, expected)
	}
	if eStatus != actualEStatus {
		t.Errorf("Unexpected exit status: %d (expected %d)", actualEStatus, eStatus)
	}
	if actual != nil {
		t.SkipNow()
	}
}

// ExitStatusErrorRE compares actual.Error() and the embeded exit status against
// expected, and triggers an error if they do not match. If actual is non-nil,
// t.SkipNow() is called as well.
func ExitStatusErrorRE(t *testing.T, expected string, eStatus int, actual error) {
	helper(t)()
	var err string
	var actualEStatus int
	if actual != nil {
		err = actual.Error()
		actualEStatus = ExitStatus(actual)
	}
	if (expected == "" && err != "") || (expected != "" && !regexp.MustCompile(expected).MatchString(err)) {
		t.Errorf("Unexpected error: %s (expected %s)", err, expected)
	}
	if actualEStatus != eStatus {
		t.Errorf("Unexpected exit status: %d (expected %d)", actualEStatus, eStatus)
	}
	if actual != nil {
		t.SkipNow()
	}
}

// FullError compares actual.Error() and the embeded HTTP and exit statuses
// against expected, and triggers an error if they do not match. If actual is
// non-nil, t.SkipNow() is called as well.
func FullError(t *testing.T, expected string, status, eStatus int, actual error) {
	helper(t)()
	var err string
	var actualStatus, actualEStatus int
	if actual != nil {
		err = actual.Error()
		actualStatus = StatusCode(actual)
		actualEStatus = ExitStatus(actual)
	}
	if expected != err {
		t.Errorf("Unexpected error: %s (expected %s)", err, expected)
	}
	if status != actualStatus {
		t.Errorf("Unexpected exit status: %d (expected %d)", actualStatus, status)
	}
	if eStatus != actualEStatus {
		t.Errorf("Unexpected exit status: %d (expected %d)", actualEStatus, eStatus)
	}
	if actual != nil {
		t.SkipNow()
	}
}

var stubRE = regexp.MustCompile(`[\s\/]`)

// Stub returns t.Name(), with whitespace and slashes converted to underscores.
func Stub(t *testing.T) string {
	// This to handle old versions of Go before the .Name() method was added.
	if n, ok := interface{}(t).(namer); ok {
		return stubRE.ReplaceAllString(n.Name(), "_")
	}
	t.Fatal("t.Name() not supported by your version of Go")
	return ""
}
