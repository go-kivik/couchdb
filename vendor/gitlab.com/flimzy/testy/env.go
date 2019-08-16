package testy

import (
	"os"
	"strings"
	"sync"
)

var envLock = new(sync.Mutex)

// RestoreEnv returns a function which restores the environment to the original
// state. It is intended to be used in conjunction with defer, to temporarily
// modify the environment during tests.
//
// Example:
//
//  func TestFoo(t *testing.T) {
//      defer RestoreEnv()()
//      os.SetEnv( ... ) // Set temporary values
//  }
func RestoreEnv() func() {
	envLock.Lock()
	env := Environ()
	return func() {
		defer envLock.Unlock()
		os.Clearenv()
		if err := SetEnv(env); err != nil {
			panic("Failed to restore environment: " + err.Error())
		}
	}
}

// Environ returns all current environment variables, parsed into a map.
func Environ() map[string]string {
	env := make(map[string]string)
	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		env[parts[0]] = parts[1]
	}
	return env
}

// SetEnv sets the environment variables contained in the map.
func SetEnv(env map[string]string) error {
	for key, val := range env {
		if err := os.Setenv(key, val); err != nil {
			return err
		}
	}
	return nil
}
