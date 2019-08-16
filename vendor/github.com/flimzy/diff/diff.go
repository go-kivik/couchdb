// Package diff provides some convenience functions for comparing text in various
// forms. It's primary use case is in automated testing.
package diff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/pmezard/go-difflib/difflib"
)

// Result is the result of a diff function. It may be nil, if the inputs were
// considered identical, or accessed via the String() method to return the
// diff. Any errors are returned as their textual representation via String()
// as well.
type Result struct {
	diff string
	err  string
}

func (r *Result) String() string {
	if r == nil {
		return ""
	}
	if r.err != "" {
		return r.err
	}
	return string(r.diff)
}

// sliceDiff expects two slices of \n-terminated strings to compare.
func sliceDiff(expected, actual []string) *Result {
	udiff := difflib.UnifiedDiff{
		A:        expected,
		FromFile: "expected",
		B:        actual,
		ToFile:   "actual",
		Context:  2,
	}
	diff, err := difflib.GetUnifiedDiffString(udiff)
	if err != nil {
		// This can only happen if a write to a byte buffer fails, so can
		// effectively be ignored, except in case of hardware failure or OOM.
		panic(err)
	}
	if diff == "" {
		return nil
	}
	return &Result{diff: diff}

}

// TextSlices compares two slices of text, treating each element as a line of
// text. Newlines are added to each element, if they are found to be missing.
func TextSlices(expected, actual []string) *Result {
	e := make([]string, len(expected))
	a := make([]string, len(actual))
	for i, str := range expected {
		e[i] = strings.TrimRight(str, "\n") + "\n"
	}
	for i, str := range actual {
		a[i] = strings.TrimRight(str, "\n") + "\n"
	}
	return sliceDiff(e, a)
}

// Text compares two strings, line-by-line, for differences.
// expected and actual must be of one of the following types:
// - string
// - []byte
// - io.Reader
func Text(expected, actual interface{}) *Result {
	exp, expErr := toText(expected)
	act, err := toText(actual)
	if err != nil {
		return &Result{err: fmt.Sprintf("[diff] actual: %s", err)}
	}
	var d *Result
	if expErr != nil {
		d = &Result{err: fmt.Sprintf("[diff] expected: %s", expErr)}
	} else {
		exp = strings.TrimSuffix(exp, "\n")
		act = strings.TrimSuffix(act, "\n")
		d = TextSlices(
			strings.SplitAfter(exp, "\n"),
			strings.SplitAfter(act, "\n"),
		)
	}
	return update(UpdateMode, expected, act, d)
}

func toText(i interface{}) (string, error) {
	switch t := i.(type) {
	case string:
		return t, nil
	case []byte:
		return string(t), nil
	case io.Reader:
		text, err := ioutil.ReadAll(t)
		return string(text), err
	case nil:
		return "", nil
	}
	return "", errors.New("input must be of type string, []byte, or io.Reader")
}

func isJSON(i interface{}) (bool, []byte, error) {
	if r, ok := i.(io.Reader); ok {
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(r); err != nil {
			return false, nil, err
		}
		return true, buf.Bytes(), nil
	}
	switch t := i.(type) {
	case []byte:
		return true, t, nil
	case json.RawMessage:
		return true, t, nil
	}
	return false, nil, nil
}

func marshal(i interface{}) ([]byte, error) {
	isJ, buf, err := isJSON(i)
	if err != nil {
		return nil, err
	}
	if isJ {
		var x interface{}
		if len(buf) > 0 {
			if e := json.Unmarshal(buf, &x); e != nil {
				return nil, e
			}
		}
		i = x
	}
	j, err := json.MarshalIndent(i, "", "    ")
	if err != nil {
		return nil, err
	}
	return j, nil
}

// AsJSON marshals two objects as JSON, then compares the output. If an input
// object is an io.Reader, it is treated as a JSON stream. If it is a []byte or
// json.RawMessage, it is treated as raw JSON. Any raw JSON source is
// unmarshaled then remarshaled with indentation for normalization and
// comparison.
func AsJSON(expected, actual interface{}) *Result {
	expectedJSON, expErr := marshal(expected)
	actualJSON, err := marshal(actual)
	if err != nil {
		return &Result{err: fmt.Sprintf("failed to marshal actual value: %s", err)}
	}
	var d *Result
	if expErr != nil {
		d = &Result{err: fmt.Sprintf("failed to marshal expected value: %s", expErr)}
	} else {
		var e, a interface{}
		_ = json.Unmarshal(expectedJSON, &e)
		_ = json.Unmarshal(actualJSON, &a)
		if reflect.DeepEqual(e, a) {
			return nil
		}
		d = Text(string(expectedJSON)+"\n", string(actualJSON)+"\n")
	}
	return update(UpdateMode, expected, string(actualJSON), d)
}

// JSON unmarshals two JSON strings, then calls AsJSON on them. As a special
// case, empty byte arrays are unmarshaled to nil.
func JSON(expected, actual []byte) *Result {
	var expectedInterface, actualInterface interface{}
	if len(expected) > 0 {
		if err := json.Unmarshal(expected, &expectedInterface); err != nil {
			return &Result{err: fmt.Sprintf("failed to unmarshal expected value: %s", err)}
		}
	}
	if len(actual) > 0 {
		if err := json.Unmarshal(actual, &actualInterface); err != nil {
			return &Result{err: fmt.Sprintf("failed to unmarshal actual value: %s", err)}
		}
	}
	return AsJSON(expectedInterface, actualInterface)
}

// Interface compares two objects with reflect.DeepEqual, and if they differ,
// it returns a diff of the spew.Dump() outputs
func Interface(expected, actual interface{}) *Result {
	if reflect.DeepEqual(expected, actual) {
		return nil
	}
	scs := spew.ConfigState{
		Indent:                  "  ",
		DisableMethods:          true,
		SortKeys:                true,
		DisablePointerAddresses: true,
		DisableCapacities:       true,
	}
	expString := scs.Sdump(expected)
	actString := scs.Sdump(actual)
	return Text(expString, actString)
}
