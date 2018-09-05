package couchdb

import (
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/errors"
)

func fullCommit(opts map[string]interface{}) (bool, error) {
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

func ifNoneMatch(opts map[string]interface{}) (string, error) {
	inm, ok := opts[OptionIfNoneMatch]
	if !ok {
		return "", nil
	}
	inmString, ok := inm.(string)
	if !ok {
		return "", errors.Statusf(kivik.StatusBadRequest, "kivik: option '%s' must be string, not %T", OptionIfNoneMatch, inm)
	}
	delete(opts, OptionIfNoneMatch)
	if inmString[0] != '"' {
		return `"` + inmString + `"`, nil
	}
	return inmString, nil
}
