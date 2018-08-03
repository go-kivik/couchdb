package chttp

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
)

// HTTPError is an error that represents an HTTP transport error.
type HTTPError struct {
	Code       int
	Reason     string `json:"reason"`
	exitStatus int
}

func (e *HTTPError) Error() string {
	if e.Reason == "" {
		return http.StatusText(e.Code)
	}
	if statusText := http.StatusText(e.Code); statusText != "" {
		return fmt.Sprintf("%s: %s", statusText, e.Reason)
	}
	return e.Reason
}

// StatusCode returns the embedded status code.
func (e *HTTPError) StatusCode() int {
	return e.Code
}

// ExitStatus returns the embedded exit status.
func (e *HTTPError) ExitStatus() int {
	return e.exitStatus
}

// ResponseError returns an error from an *http.Response.
func ResponseError(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	httpErr := &HTTPError{
		exitStatus: ExitNotRetrieved,
	}
	if resp.Request.Method != "HEAD" && resp.ContentLength != 0 {
		if ct, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); ct == typeJSON {
			_ = json.NewDecoder(resp.Body).Decode(httpErr)
		}
	}
	httpErr.Code = resp.StatusCode
	return httpErr
}

type curlError struct {
	curlStatus int
	httpStatus int
	error
}

func (e *curlError) ExitStatus() int {
	return e.curlStatus
}

func (e *curlError) StatusCode() int {
	return e.httpStatus
}

func fullError(httpStatus, curlStatus int, err error) error {
	return &curlError{
		curlStatus: curlStatus,
		httpStatus: httpStatus,
		error:      err,
	}
}
