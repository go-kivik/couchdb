// +build !go1.9

package testy

import "testing"

func helper(t *testing.T) func() {
	return func() {}
}
