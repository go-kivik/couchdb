package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/util"
	"github.com/flimzy/kivik/errors"
	"github.com/go-kivik/couchdb/chttp"
)

var findNotImplemented = errors.Status(kivik.StatusNotImplemented, "kivik: Find interface not implemented prior to CouchDB 2.0.0")

func (d *db) CreateIndex(ctx context.Context, ddoc, name string, index interface{}) error {
	if d.client.Compat == CompatCouch16 {
		return findNotImplemented
	}
	indexObj, err := deJSONify(index)
	if err != nil {
		return err
	}
	parameters := struct {
		Index interface{} `json:"index"`
		Ddoc  string      `json:"ddoc,omitempty"`
		Name  string      `json:"name,omitempty"`
	}{
		Index: indexObj,
		Ddoc:  ddoc,
		Name:  name,
	}
	body := &bytes.Buffer{}
	if err = json.NewEncoder(body).Encode(parameters); err != nil {
		return errors.WrapStatus(kivik.StatusBadRequest, err)
	}
	_, err = d.Client.DoError(ctx, kivik.MethodPost, d.path("_index", nil), &chttp.Options{Body: body})
	return err
}

func (d *db) GetIndexes(ctx context.Context) ([]driver.Index, error) {
	if d.client.Compat == CompatCouch16 {
		return nil, findNotImplemented
	}
	var result struct {
		Indexes []driver.Index `json:"indexes"`
	}
	_, err := d.Client.DoJSON(ctx, kivik.MethodGet, d.path("_index", nil), nil, &result)
	return result.Indexes, err
}

func (d *db) DeleteIndex(ctx context.Context, ddoc, name string) error {
	if d.client.Compat == CompatCouch16 {
		return findNotImplemented
	}
	if ddoc == "" {
		return missingArg("ddoc")
	}
	if name == "" {
		return missingArg("name")
	}
	path := fmt.Sprintf("_index/%s/json/%s", ddoc, name)
	_, err := d.Client.DoError(ctx, kivik.MethodDelete, d.path(path, nil), nil)
	return err
}

func (d *db) Find(ctx context.Context, query interface{}) (driver.Rows, error) {
	if d.client.Compat == CompatCouch16 {
		return nil, findNotImplemented
	}
	body, err := toJSON(query)
	if err != nil {
		return nil, err
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodPost, d.path("_find", nil), &chttp.Options{Body: body})
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return newRows(resp.Body), nil
}

type queryPlan struct {
	DBName   string                 `json:"dbname"`
	Index    map[string]interface{} `json:"index"`
	Selector map[string]interface{} `json:"selector"`
	Options  map[string]interface{} `json:"opts"`
	Limit    int64                  `json:"limit"`
	Skip     int64                  `json:"skip"`
	Fields   fields                 `json:"fields"`
	Range    map[string]interface{} `json:"range"`
}

type fields []interface{}

func (f *fields) UnmarshalJSON(data []byte) error {
	if string(data) == `"all_fields"` {
		return nil
	}
	var i []interface{}
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	newFields := make([]interface{}, len(i))
	copy(newFields, i)
	*f = newFields
	return nil
}

func (d *db) Explain(ctx context.Context, query interface{}) (*driver.QueryPlan, error) {
	if d.client.Compat == CompatCouch16 {
		return nil, findNotImplemented
	}
	body, err := util.ToJSON(query)
	if err != nil {
		return nil, err
	}
	var plan queryPlan
	_, err = d.Client.DoJSON(ctx, kivik.MethodPost, d.path("_explain", nil), &chttp.Options{Body: body}, &plan)
	if err != nil {
		return nil, err
	}
	return &driver.QueryPlan{
		DBName:   plan.DBName,
		Index:    plan.Index,
		Selector: plan.Selector,
		Options:  plan.Options,
		Limit:    plan.Limit,
		Skip:     plan.Skip,
		Fields:   plan.Fields,
		Range:    plan.Range,
	}, nil
}
