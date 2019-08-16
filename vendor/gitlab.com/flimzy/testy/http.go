package testy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ResponseHandler wraps an existing http.Response, to be served as a
// standard http.Handler
type ResponseHandler struct {
	*http.Response
}

var _ http.Handler = &ResponseHandler{}

func (h *ResponseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for header, values := range h.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	if h.StatusCode != 0 {
		w.WriteHeader(h.StatusCode)
	}
	if h.Body != nil {
		defer h.Body.Close() // nolint: errcheck
		_, _ = io.Copy(w, h.Body)
	}
}

// ServeResponse starts a test HTTP server that serves r.
func ServeResponse(r *http.Response) *httptest.Server {
	return httptest.NewServer(&ResponseHandler{r})
}

// RequestValidator is a function that takes a *http.Request for validation.
type RequestValidator func(*testing.T, *http.Request)

// ValidateRequest returns a middleware that calls fn(), to validate the HTTP
// request, before continuing. An error returned by fn() will result in the
// addition of an X-Error header, a 400 status, and the error added to the
// body of the response.
func ValidateRequest(t *testing.T, fn RequestValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn(t, r)
			next.ServeHTTP(w, r)
		})
	}
}

// ServeResponseValidator wraps a ResponseHandler with ValidateRequest
// middleware for a complete response-serving, request-validating test server.
func ServeResponseValidator(t *testing.T, r *http.Response, fn RequestValidator) *httptest.Server {
	mw := ValidateRequest(t, fn)
	return httptest.NewServer(mw(&ResponseHandler{r}))
}

// HTTPResponder is a function which intercepts and responds to an HTTP request.
type HTTPResponder func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = HTTPResponder(nil)

// RoundTrip satisfies the http.RoundTripper interface
func (t HTTPResponder) RoundTrip(r *http.Request) (*http.Response, error) {
	return t(r)
}

// HTTPClient returns a customized *http.Client, which passes the request to
// fn, rather than to the network.
func HTTPClient(fn HTTPResponder) *http.Client {
	return &http.Client{Transport: fn}
}
