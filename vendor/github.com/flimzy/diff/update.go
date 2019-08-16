package diff

import (
	"fmt"
	"os"
)

func update(updateMode bool, expected interface{}, actual string, d *Result) *Result {
	if d == nil || !updateMode {
		return d
	}
	expectedFile, ok := expected.(*File)
	if !ok {
		return d
	}
	file, err := os.OpenFile(expectedFile.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666) // nolint: gas
	if err != nil {
		return &Result{err: fmt.Sprintf("Update failed: %s", err)}
	}
	defer file.Close() // nolint: errcheck
	if _, e := file.WriteString(actual); e != nil {
		return &Result{err: fmt.Sprintf("Update failed: %s", e)}
	}
	return nil
}
