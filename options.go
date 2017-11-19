package couchdb

import (
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/errors"
)

func fullCommit(fullCommit bool, opts map[string]interface{}) (bool, error) {
	for _, key := range []string{optionForceCommit, OptionFullCommit} {
		fc, ok := opts[key]
		if ok {
			fcBool, ok := fc.(bool)
			if !ok {
				return false, errors.Statusf(kivik.StatusBadRequest, "kivik: option '%s' must be bool, not %T", key, fc)
			}
			fullCommit = fcBool
			delete(opts, key)
		}
	}
	return fullCommit, nil
}
