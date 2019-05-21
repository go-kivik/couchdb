package couchdb

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/flimzy/testy"
)

func TestCancelableReadCloser(t *testing.T) {
	t.Run("no cancelation", func(t *testing.T) {
		t.Parallel()
		rc := newCancelableReadCloser(
			context.Background(),
			ioutil.NopCloser(strings.NewReader("foo")),
		)
		result, err := ioutil.ReadAll(rc)
		testy.Error(t, "", err)
		if string(result) != "foo" {
			t.Errorf("Unexpected result: %s", string(result))
		}
	})
	t.Run("pre-canceled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		rc := newCancelableReadCloser(
			ctx,
			ioutil.NopCloser(strings.NewReader("foo")),
		)
		result, err := ioutil.ReadAll(rc)
		testy.Error(t, "context canceled", err)
		if string(result) != "" {
			t.Errorf("Unexpected result: %s", string(result))
		}
	})
	t.Run("canceled mid-flight", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()
		r := io.MultiReader(
			strings.NewReader("foo"),
			testy.DelayReader(time.Second),
			strings.NewReader("bar"),
		)
		rc := newCancelableReadCloser(
			ctx,
			ioutil.NopCloser(r),
		)
		result, err := ioutil.ReadAll(rc)
		testy.Error(t, "context deadline exceeded", err)
		if string(result) != "" {
			t.Errorf("Unexpected result: %s", string(result))
		}
	})
	t.Run("read error, not canceled", func(t *testing.T) {
		t.Parallel()
		rc := newCancelableReadCloser(
			context.Background(),
			ioutil.NopCloser(testy.ErrorReader("foo", errors.New("read err"))),
		)
		result, err := ioutil.ReadAll(rc)
		testy.Error(t, "read err", err)
		if string(result) != "" {
			t.Errorf("Unexpected result: %s", string(result))
		}
	})
}
