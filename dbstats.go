package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"reflect"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/errors"
)

func (d *db) Stats(ctx context.Context) (*driver.DBStats, error) {
	res, err := d.Client.DoReq(ctx, kivik.MethodGet, d.dbName, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() // nolint: errcheck
	if err = chttp.ResponseError(res); err != nil {
		return nil, err
	}
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.WrapStatus(kivik.StatusNetworkError, err)
	}
	result := struct {
		driver.DBStats
		Sizes struct {
			File     int64 `json:"file"`
			External int64 `json:"external"`
			Active   int64 `json:"active"`
		} `json:"sizes"`
		UpdateSeq json.RawMessage `json:"update_seq"`
	}{}
	if err := json.Unmarshal(resBody, &result); err != nil {
		return nil, errors.WrapStatus(kivik.StatusBadResponse, err)
	}
	stats := &result.DBStats
	if result.Sizes.File > 0 {
		stats.DiskSize = result.Sizes.File
	}
	if result.Sizes.External > 0 {
		stats.ExternalSize = result.Sizes.External
	}
	if result.Sizes.Active > 0 {
		stats.ActiveSize = result.Sizes.Active
	}
	stats.UpdateSeq = string(bytes.Trim(result.UpdateSeq, `"`))
	// Reflection is used to preserve backward compatibility with Kivik stable
	// 1.7.3 and unstable prior to 25 June 2018. The reflection hack can be
	// removed at some point in the reasonable future.
	if v := reflect.ValueOf(stats).Elem().FieldByName("RawResponse"); v.CanSet() {
		v.Set(reflect.ValueOf(resBody))
	}
	return stats, nil
}
