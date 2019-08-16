// +build go1.10

package kiviktest

import (
	"io"
	"os"
	"regexp"
	"testing"
)

// testDeps is a copy of testing.testDeps
type testDeps interface {
	ImportPath() string
	MatchString(pat, str string) (bool, error)
	StartCPUProfile(io.Writer) error
	StopCPUProfile()
	StartTestLog(io.Writer)
	StopTestLog() error
	WriteHeapProfile(io.Writer) error
	WriteProfileTo(string, io.Writer, int) error
}

type deps struct{}

var _ testDeps = &deps{}

func (d *deps) MatchString(pat, str string) (bool, error)         { return regexp.MatchString(pat, str) }
func (d *deps) StartCPUProfile(_ io.Writer) error                 { return nil }
func (d *deps) StopCPUProfile()                                   {}
func (d *deps) WriteHeapProfile(_ io.Writer) error                { return nil }
func (d *deps) WriteProfileTo(_ string, _ io.Writer, _ int) error { return nil }
func (d *deps) ImportPath() string                                { return "" }
func (d *deps) StartTestLog(io.Writer)                            {}
func (d *deps) StopTestLog() error                                { return nil }

func mainStart(tests []testing.InternalTest) {
	m := testing.MainStart(&deps{}, tests, nil, nil)
	os.Exit(m.Run())
}
