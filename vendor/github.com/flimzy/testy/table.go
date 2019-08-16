package testy

import (
	"fmt"
	"reflect"
	"testing"
)

// Generator should return a test.
type Generator func(*testing.T) interface{}

// Table is a table of one or more tests to run against a single test function.
type Table struct {
	gens    map[string]Generator
	cleanup []func(*testing.T)
}

// NewTable returns a new Table instance. As a single argument, it may take a
// map of named test generators.
func NewTable(generators ...map[string]Generator) *Table {
	tb := &Table{
		gens:    make(map[string]Generator),
		cleanup: make([]func(*testing.T), 0),
	}
	for _, g := range generators {
		for k, v := range g {
			tb.Add(k, v)
		}
	}
	return tb
}

// Add adds a single named test to the table.
// test may be of the following types:
//
//    - interface{}
//    - func() interface{}
//    - func(*testing.T) interface{}
func (tb *Table) Add(name string, test interface{}) {
	if _, ok := tb.gens[name]; ok {
		panic("Add(): Test " + name + " already defined.")
	}
	var gen func(*testing.T) interface{}
	switch typedTest := test.(type) {
	case func() interface{}:
		gen = func(_ *testing.T) interface{} {
			return typedTest()
		}
	case func(*testing.T) interface{}:
		gen = typedTest
	default:
		if reflect.TypeOf(test).Kind() == reflect.Func {
			panic(fmt.Sprintf("Test generator must be of type func(*testing.T) interface{} or func() interface{}, got %T", test))
		}
		gen = func(_ *testing.T) interface{} {
			return test
		}
	}
	tb.gens[name] = gen
}

// Cleanup takes a single function to be called after all Table tests have been
// executed. fn must match one of the following signatures:
//
//    - func()
//    - func() error
//    - func(*testing.T)
func (tb *Table) Cleanup(fn interface{}) {
	var cleanup func(*testing.T)
	switch typedFn := fn.(type) {
	case func():
		cleanup = func(_ *testing.T) { typedFn() }
	case func(*testing.T):
		cleanup = typedFn
	case func() error:
		cleanup = func(t *testing.T) {
			helper(t)()
			if e := typedFn(); e != nil {
				t.Error(e)
			}
		}
	default:
		panic("Cleanup function must be func(), func() error, or func(*testing.T)")
	}
	tb.cleanup = append(tb.cleanup, cleanup)
}

func (tb *Table) doCleanup(t *testing.T) {
	for _, fn := range tb.cleanup {
		fn(t)
	}
}

// Run cycles through the defined tests, passing them one at a time to testFn.
// testFn must be a function which takes two arguments: *testing.T, and an
// arbitrary type, which must match the return value of the Generator functions.
func (tb *Table) Run(t *testing.T, testFn interface{}) {
	defer tb.doCleanup(t)
	testFnT := reflect.TypeOf(testFn)
	if testFnT.Kind() != reflect.Func {
		panic("testFn must be a function")
	}
	if testFnT.NumIn() != 2 || testFnT.In(0) != reflect.TypeOf(&testing.T{}) {
		panic("testFn must be of the form func(*testing.T, **)")
	}
	testType := reflect.TypeOf(testFn).In(1)
	testFnV := reflect.ValueOf(testFn)
	for name, genFn := range tb.gens {
		t.Run(name, func(t *testing.T) {
			helper(t)()
			test := genFn(t)
			if reflect.TypeOf(test) != testType {
				t.Fatalf("Test generator returned wrong type. Have %T, want %s", test, testType.Name())
			}
			_ = testFnV.Call([]reflect.Value{reflect.ValueOf(t), reflect.ValueOf(test)})
		})
	}
}
