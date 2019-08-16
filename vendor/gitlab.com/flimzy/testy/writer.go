package testy

import (
	"io"
	"strings"
)

type errWriter struct {
	count int
	err   error
}

func (w *errWriter) Write(p []byte) (int, error) {
	c := len(p)
	if c >= w.count {
		return w.count, w.err
	}
	w.count -= c
	return c, nil
}

// ErrorWriter returns a new io.Writer, which will accept count bytes
// (discarding them), then return err as an error.
func ErrorWriter(count int, err error) io.Writer {
	return &errWriter{count: count, err: err}
}

type errReader struct {
	r   io.Reader
	err error
}

func (r *errReader) Read(p []byte) (int, error) {
	c, err := r.r.Read(p)
	if err == io.EOF {
		err = r.err
	}
	return c, err
}

// ErrorReader returns an io.Reader which behaves the same as
// strings.NewReader(s), except that err will be returned at the end, rather
// than io.EOF.
func ErrorReader(s string, err error) io.Reader {
	return &errReader{r: strings.NewReader(s), err: err}
}
