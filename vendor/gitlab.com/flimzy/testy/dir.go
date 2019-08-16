package testy

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/otiai10/copy"
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
//         var tmpdir string
//         defer testy.TempDir(t, &tmpdir)()
//         /*  here you can use tmpdir  */
//     })
//
//  Or:
//
//      tests := testy.NewTable()
//      tests.Add("foo", func(t *testing.T) interface{} {
//          var tmpdir string
//          tests.Cleanup(testy.TempDir(t, &tmpdir))
//          // ...
//       })
func TempDir(t *testing.T, dirname *string) func() {
	helper(t)()
	dir := tempDir(t)
	*dirname = dir
	return func() {
		_ = os.RemoveAll(dir)
	}
}

func tempDir(t *testing.T) string {
	t.Helper()
	tmpdir, err := ioutil.TempDir("", testPrefix(t))
	if err != nil {
		t.Fatal(err)
	}
	return tmpdir
}

func testPrefix(t *testing.T) string {
	// This to handle old versions of Go before the .Name() method was added.
	if n, ok := interface{}(t).(namer); ok {
		return strings.Replace(n.Name(), string(os.PathSeparator), "_", -1) + "-"
	}
	return "testy-"
}

// JSONDir implements a json marshaler on the filesystem path specified by the
// Path field. All other fields are optional, with reasonable defaults for most
// use cases.
type JSONDir struct {
	// Path is the filesystem path to be serialized.
	Path string
	// NoMD5Sum suppresses the output of the MD5 sum in output.
	NoMD5Sum bool
	// NoSize suppresses the file size output.
	NoSize bool
	// FileContent turns on the inclusion of file contents.
	FileContent bool
	// MaxContentSize sets the limit for included content. Files larger than
	// this will be truncated in their output.
	MaxContentSize int
}

type dirEntry struct {
	MD5     string `json:"md5,omitempty"`
	Size    int64  `json:"size,omitempty"`
	Content string `json:"content,omitempty"`
}

// MarshalJSON satisfies the json.Marshaler interface.
func (d JSONDir) MarshalJSON() ([]byte, error) {
	base := d.Path
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	dir := make(map[string]dirEntry)
	err := filepath.Walk(d.Path, func(fullpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relpath := strings.TrimPrefix(fullpath, base)
		entry := dirEntry{}
		if !d.NoMD5Sum {
			entry.MD5, err = md5sum(fullpath)
			if err != nil {
				return err
			}
		}
		if !d.NoSize {
			entry.Size = info.Size()
		}
		if d.FileContent {
			content, err := d.fileContent(fullpath)
			if err != nil {
				return err
			}
			entry.Content = string(content)
		}
		dir[relpath] = entry
		return nil
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(dir)
}

func (d JSONDir) fileContent(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !isASCII(content) {
		dst := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
		base64.StdEncoding.Encode(dst, content)
		content = dst
	}
	if d.MaxContentSize > 0 && len(content) > d.MaxContentSize {
		content = content[:d.MaxContentSize]
	}
	return string(content), nil
}

// borrowed from https://stackoverflow.com/a/53069799/13860
func isASCII(s []byte) bool {
	ct := http.DetectContentType(s)
	if strings.Contains(ct, ";") {
		ct = ct[:strings.Index(ct, ";")]
	}
	switch ct {
	case "text/plain", "text/html", "application/json":
		return true
	}
	switch ct[:strings.Index(ct, "/")] {
	case "image", "audio", "archive", "application":
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func md5sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() // nolint: errcheck

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CopyTempDir recursively copies the contents of path to a new temporary dir
// whose path is returned. The depth argument controls how deeply path is
// placed into the temp dir. Examples:
//
//  copyDir(t, "/foo/bar/baz", 0) // copies /foo/bar/baz/* to /tmp-XXX/*
//  copyDir(t, "/foo/bar/baz", 1) // copies /foo/bar/baz/* to /tmp-XXX/baz/*
//  copyDir(t, "/foo/bar/baz", 3) // copies /foo/bar/baz/* to /tmp-XXX/foo/bar/baz/*
func CopyTempDir(t *testing.T, source string, depth int) string { // nolint: unparam
	t.Helper()
	tmpdir := tempDir(t)
	target := tmpdir
	if depth > 0 {
		parts := strings.Split(source, string(filepath.Separator))
		if len(parts) < depth {
			t.Fatalf("Depth of %d specified, but path only has %d parts", depth, len(parts))
		}
		target = filepath.Join(append([]string{tmpdir}, parts[len(parts)-depth:]...)...)
		if err := os.MkdirAll(target, 0777); err != nil {
			t.Fatal(err)
		}
	}
	if err := copy.Copy(source, target); err != nil {
		t.Fatal(err)
	}
	return tmpdir
}
