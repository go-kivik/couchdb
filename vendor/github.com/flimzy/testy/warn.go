package testy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Warn outputs the formatted text to STDERR, prefixed with some useful
// debugging info. It is named after perl's 'warn' builtin. The output prefix
// is in the following format:
//
// [PID/GID file:lineno]
func Warn(format string, args ...interface{}) {
	_, _ = Warnf(os.Stderr, format, args...)
}

// Swarn returns a Warn-formatted string.
func Swarn(format string, args ...interface{}) string {
	return warn() + fmt.Sprintf(format, args...)
}

// Warnf warns to the specified io.Writer.
func Warnf(f io.Writer, format string, args ...interface{}) (int, error) {
	return f.Write([]byte(Swarn(format, args...)))
}

func warn() string {
	var file string
	var line int
	for skip := 2; skip < 5; skip++ {
		_, file, line, _ = runtime.Caller(skip)
		if !strings.HasSuffix(file, "flimzy/testy/warn.go") {
			break
		}
	}
	return fmt.Sprintf("[%d/%d %s:%d] ", os.Getpid(), getGID(), file, line)
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
