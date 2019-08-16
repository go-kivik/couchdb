package testy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Sadly, since os.Stdin, os.Stdout, and os.Stderr are truly global variables,
// a global lock is the only option here.
var globalLock sync.Mutex

// RedirIO runs fn, with stdin redirected from in, and anything sent to stdout
// or stderr returned.
func RedirIO(in io.Reader, fn func()) (stdout, stderr io.Reader) {
	globalLock.Lock()
	defer globalLock.Unlock()
	cleanupIn := replaceInput(&os.Stdin, in)
	tmpout, cleanupOut := replaceOutput(&os.Stdout)
	tmperr, cleanupErr := replaceOutput(&os.Stderr)

	defer func() {
		if r := recover(); r != nil {
			fmt.Print(tmpout)
			_, _ = fmt.Fprint(os.Stderr, tmperr)
			panic(r)
		}
	}()
	defer cleanupIn()
	defer cleanupOut()
	defer cleanupErr()

	fn()

	return tmpout, tmperr
}

func replaceInput(old **os.File, in io.Reader) func() {
	if in == nil {
		in = strings.NewReader("")
	}
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	go func() {
		if _, err := io.Copy(w, in); err != nil {
			panic(err)
		}
		w.Close()
	}()
	orig := *old
	*old = r
	return func() {
		*old = orig
	}
}

func replaceOutput(old **os.File) (io.Reader, func()) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	new := &bytes.Buffer{}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if _, err := io.Copy(new, r); err != nil {
			panic(err)
		}
		wg.Done()
	}()
	orig := *old
	*old = w
	return new, func() {
		w.Close()
		wg.Wait()
		*old = orig
	}
}
