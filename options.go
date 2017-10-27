package couchdb

import (
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/errors"
)

func forceCommit(opts map[string]interface{}) (bool, error) {
	fc, ok := opts[OptionFullCommit]
	if !ok {
		return false, nil
	}
	fcBool, ok := fc.(bool)
	if !ok {
		return false, errors.Statusf(kivik.StatusBadRequest, "kivik: option '%s' must be bool, not %T", OptionFullCommit, fc)
	}
	delete(opts, OptionFullCommit)
	return fcBool, nil
}
