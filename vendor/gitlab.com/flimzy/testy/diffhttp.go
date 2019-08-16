package testy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
)

// DiffHTTPRequest compares the metadata and bodies of the two HTTP requests,
// and returns the difference.
// Inputs must be of the type *http.Request, or of one of the following types,
// in which case the input is interpreted as a raw HTTP request.
// - io.Reader
// - string
// - []byte
func DiffHTTPRequest(expected, actual interface{}) *Diff {
	expDump, expErr := dumpRequest(expected)
	actDump, err := dumpRequest(actual)
	if err != nil {
		return &Diff{err: fmt.Sprintf("Failed to dump actual request: %s", err)}
	}
	var d *Diff
	if expErr != nil {
		d = &Diff{err: fmt.Sprintf("Failed to dump expected request: %s", expErr)}
	} else {
		d = DiffText(string(expDump), string(actDump))
	}
	return update(UpdateMode, expected, string(actDump), d)
}

func toRequest(i interface{}) (*http.Request, error) {
	var r io.Reader
	switch t := i.(type) {
	case *http.Request:
		return t, nil
	case io.Reader:
		r = t
	case string:
		r = strings.NewReader(t)
	case []byte:
		r = bytes.NewReader(t)
	default:
		return nil, fmt.Errorf("Unable to convert %T to *http.Request", i)
	}
	return http.ReadRequest(bufio.NewReader(r))
}

func dumpRequest(i interface{}) ([]byte, error) {
	if i == nil {
		return nil, nil
	}
	req, err := toRequest(i)
	if err != nil {
		return nil, err
	}
	return httputil.DumpRequest(req, true)
}

// DiffHTTPResponse compares the metadata and bodies of the two HTTP responses,
// and returns the difference.
// Inputs must be of the type *http.Response, or of one of the following types,
// in which case the input is interpreted as a raw HTTP response.
// - io.Reader
// - string
// - []byte
func DiffHTTPResponse(expected, actual interface{}) *Diff {
	expDump, expErr := dumpResponse(expected)
	actDump, err := dumpResponse(actual)
	if err != nil {
		return &Diff{err: fmt.Sprintf("Failed to dump actual response: %s", err)}
	}
	var d *Diff
	if expErr != nil {
		d = &Diff{err: fmt.Sprintf("Failed to dump expected response: %s", expErr)}
	} else {
		d = DiffText(string(expDump), string(actDump))
	}
	return update(UpdateMode, expected, string(actDump), d)
}

func toResponse(i interface{}) (*http.Response, error) {
	var r io.Reader
	switch t := i.(type) {
	case *http.Response:
		return t, nil
	case io.Reader:
		r = t
	case string:
		r = strings.NewReader(t)
	case []byte:
		r = bytes.NewReader(t)
	default:
		return nil, fmt.Errorf("Unable to convert %T to *http.Response", i)
	}
	return http.ReadResponse(bufio.NewReader(r), nil)
}

func dumpResponse(i interface{}) ([]byte, error) {
	if i == nil {
		return nil, nil
	}
	res, err := toResponse(i)
	if err != nil {
		return nil, err
	}
	return httputil.DumpResponse(res, true)
}
