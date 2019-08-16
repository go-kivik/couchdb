package testy

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

type namer interface {
	Name() string
}

// TempDir creates a temporary directory, assigning the name to dirname, and
// returning a cleanup function, which will remove the tempdir and all contents.
// An error during creation will be passed to t.Fatal(). An error during
// cleanup will be ignored.
//
// It is meant to be run as so:
//
//     t.Run(func(t *testing.T) {
//         tmpDir := new(string)
//         defer testy.TempDir(t, tmpDir)()
//         /*  here you can use tmpDir  */
//     })
func TempDir(t *testing.T, dirname *string) func() {
	helper(t)()
	dir, err := ioutil.TempDir("", testPrefix(t))
	if err != nil {
		t.Fatal(err)
	}
	*dirname = dir
	return func() {
		_ = os.RemoveAll(dir)
	}
}

func testPrefix(t *testing.T) string {
	// This to handle old versions of Go before the .Name() method was added.
	if n, ok := interface{}(t).(namer); ok {
		return strings.Replace(n.Name(), string(os.PathSeparator), "_", -1) + "-"
	}
	return "testy-"
}
