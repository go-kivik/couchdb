package testy

import (
	"io"
	"time"
)

type delayReader struct {
	delay time.Duration
}

var _ io.Reader = &delayReader{}

func (r *delayReader) Read(_ []byte) (int, error) {
	time.Sleep(r.delay)
	return 0, io.EOF
}

// DelayReader returns an io.Reader that never returns any data, but will
// delay before return io.EOF. It is intended to be used in conjunction with
// io.MultiReader to construct test scenarios. Each call to Read() will delay,
// then return (0, io.EOF), so it is safe to re-use the return value.
func DelayReader(delay time.Duration) io.Reader {
	return &delayReader{delay: delay}
}
