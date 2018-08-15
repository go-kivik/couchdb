package chttp

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
)

type clientTraceContextKey struct{}

// ContextClientTrace returns the ClientTrace associated with the
// provided context. If none, it returns nil.
func ContextClientTrace(ctx context.Context) *ClientTrace {
	trace, _ := ctx.Value(clientTraceContextKey{}).(*ClientTrace)
	return trace
}

// ClientTrace is a set of hooks to run at various stages of an outgoing
// HTTP request. Any particular hook may be nil. Functions may be
// called concurrently from different goroutines and some may be called
// after the request has completed or failed.
type ClientTrace struct {
	// HTTPResponse returns a cloe of the *http.Response received from the
	// server, with the body set to nil. If you need the body, use the more
	// expensive HTTPResponseBody
	HTTPResponse func(*http.Response)

	// HTTPResponseBody returns a clone of the *http.Response received from the
	// server, with the body cloned. This can be expensive for responses
	// with large bodies.
	HTTPResponseBody func(*http.Response)
}

// WithClientTrace returns a new context based on the provided parent
// ctx. HTTP client requests made with the returned context will use
// the provided trace hooks, in addition to any previous hooks
// registered with ctx. Any hooks defined in the provided trace will
// be called first.
func WithClientTrace(ctx context.Context, trace *ClientTrace) context.Context {
	if trace == nil {
		panic("nil trace")
	}
	return context.WithValue(ctx, clientTraceContextKey{}, trace)
}

func (t *ClientTrace) httpResponse(r *http.Response) {
	if t.HTTPResponse == nil {
		return
	}
	clone := &http.Response{}
	*clone = *r
	clone.Body = nil
	t.HTTPResponse(clone)
}

func (t *ClientTrace) httpResponseBody(r *http.Response) {
	if t.HTTPResponseBody == nil {
		return
	}
	clone := &http.Response{}
	*clone = *r
	rBody := r.Body
	body, readErr := ioutil.ReadAll(rBody)
	closeErr := rBody.Close()
	r.Body = newReplay(body, readErr, closeErr)
	clone.Body = newReplay(body, readErr, closeErr)
	t.HTTPResponseBody(clone)
}

func newReplay(body []byte, readErr, closeErr error) io.ReadCloser {
	if readErr == nil && closeErr == nil {
		return ioutil.NopCloser(bytes.NewReader(body))
	}
	return &replayReadCloser{
		Reader:   ioutil.NopCloser(bytes.NewReader(body)),
		readErr:  readErr,
		closeErr: closeErr,
	}
}

// replayReadCloser replays read and close errors
type replayReadCloser struct {
	io.Reader
	readErr  error
	closeErr error
}

func (r *replayReadCloser) Read(p []byte) (int, error) {
	c, err := r.Reader.Read(p)
	if err == io.EOF && r.readErr != nil {
		err = r.readErr
	}
	return c, err
}

func (r *replayReadCloser) Close() error {
	return r.closeErr
}
