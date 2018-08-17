package chttp

import (
	"io"
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
		t.Errorf("Unexpected status code: %d (expected %d) [%s]", actualStatus, status, err)
	}
	if eStatus != actualExitStatus {
		t.Errorf("Unexpected exit status: %d (expected %d) [%s]", actualExitStatus, eStatus, err)
	}
	if actual != nil {
		t.SkipNow()
	}
}

type errReader struct {
	io.Reader
	err error
}

func (r *errReader) Read(p []byte) (int, error) {
	c, err := r.Reader.Read(p)
	if err == io.EOF {
		err = r.err
	}
	return c, err
}

type errCloser struct {
	io.Reader
	err error
}

func (r *errCloser) Close() error {
	return r.err
}
