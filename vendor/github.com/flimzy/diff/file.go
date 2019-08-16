package diff

import (
	"io"
	"os"
)

// File converts a file into an io.Reader.
// When UpdateMode is true, a dected difference will cause File to be
// overwritten with the actual value, when File is the expected value.
type File struct {
	Path string
	r    io.Reader
	done bool
}

var _ io.Reader = &File{}

func (f *File) Read(p []byte) (int, error) {
	if f.done {
		return 0, io.EOF
	}
	if f.r == nil {
		var err error
		if f.r, err = os.Open(f.Path); err != nil {
			return 0, err
		}
	}
	n, err := f.r.Read(p)
	f.done = err != nil
	return n, err
}
