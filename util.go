package couchdb

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/errors"
)

// deJSONify unmarshals a string, []byte, or json.RawMessage. All other types
// are returned as-is.
func deJSONify(i interface{}) (interface{}, error) {
	var data []byte
	switch t := i.(type) {
	case string:
		data = []byte(t)
	case []byte:
		data = t
	case json.RawMessage:
		data = []byte(t)
	default:
		return i, nil
	}
	var x interface{}
	err := json.Unmarshal(data, &x)
	return x, errors.WrapStatus(kivik.StatusBadRequest, err)
}

// toJSON converts a string, []byte, json.RawMessage, or an arbitrary type into
// an io.Reader of JSON marshaled data.
func toJSON(i interface{}) (io.Reader, error) {
	switch t := i.(type) {
	case string:
		return strings.NewReader(t), nil
	case []byte:
		return bytes.NewReader(t), nil
	case json.RawMessage:
		return bytes.NewReader(t), nil
	default:
		buf := &bytes.Buffer{}
		err := json.NewEncoder(buf).Encode(i)
		return buf, errors.WrapStatus(kivik.StatusBadRequest, err)
	}
}
