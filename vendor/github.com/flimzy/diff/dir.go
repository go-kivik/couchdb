package diff

import (
	"crypto/md5" // nolint: gas
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
)

// DirChecksum compares the checksum of the contents of dir against the checksums
// in expected. Expected should be a map of all files expected in the directory,
// with the full path and filename as key, and the md5 sum as the value.
func DirChecksum(expected map[string]string, dir string) *Result {
	actual, err := checkDir(dir, false)
	if err != nil {
		return &Result{err: err.Error()}
	}
	return Interface(expected, actual)
}

// DirFullCheck compares the checksum of the contents of dir against the checksums
// in expected. Expected should be a map of all files expected in the directory,
// with the full path and filename as key, and the md5 sum, mode, and ownership
// as the value.
func DirFullCheck(expected map[string]string, dir string) *Result {
	actual, err := checkDir(dir, true)
	if err != nil {
		return &Result{err: err.Error()}
	}
	return Interface(expected, actual)
}

func checkDir(dir string, full bool) (map[string]string, error) {
	result := make(map[string]string)
	err := recurseDir(result, full, []string{dir})
	return result, err
}

func recurseDir(result map[string]string, full bool, parents []string) error {
	dir := strings.Join(parents, "/")
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	files, err := f.Readdir(0)
	if err != nil {
		return err
	}
	for _, f := range files {
		relName := relativeName(parents, f)
		h, err := hash(dir, f, full)
		if err != nil {
			return err
		}
		result[relName] = h
		if f.IsDir() {
			if err := recurseDir(result, full, append(parents, f.Name())); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

func relativeName(parents []string, f os.FileInfo) string {
	parts := append(parents[1:], f.Name())
	name := strings.Join(parts, "/")
	if f.IsDir() {
		return name + "/"
	}
	return name
}

func hash(dir string, f os.FileInfo, full bool) (string, error) {
	var hash string
	if f.IsDir() {
		hash = "<dir>"
	} else {
		content, err := os.Open(dir + "/" + f.Name())
		if err != nil {
			return "", err
		}
		h := md5.New() // nolint: gas
		if _, err := io.Copy(h, content); err != nil {
			return "", err
		}
		if err := content.Close(); err != nil {
			return "", err
		}
		hash = hex.EncodeToString(h.Sum([]byte{}))
	}
	if full {
		return fmt.Sprintf("%04o %s %s", f.Mode()&0xfff, owner(f), hash), nil
	}
	return hash, nil
}

func owner(f os.FileInfo) string {
	switch t := f.Sys().(type) {
	case *syscall.Stat_t:
		return fmt.Sprintf("%d.%d", t.Uid, t.Gid)
	default:
		return "---"
	}
}
