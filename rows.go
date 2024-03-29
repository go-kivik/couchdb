// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync/atomic"

	"github.com/go-kivik/kivik/v4/driver"
)

type rowsMeta struct {
	offset    int64
	totalRows int64
	updateSeq sequenceID
	warning   string
	bookmark  string
}

type rows struct {
	*iter
	meta *rowsMeta
}

var _ driver.Rows = &rows{}

type rowsMetaParser struct{}

func (p *rowsMetaParser) parseMeta(i interface{}, dec *json.Decoder, key string) error {
	meta := i.(*rowsMeta)
	return meta.parseMeta(key, dec)
}

type rowParser struct {
	rowsMetaParser
}

var _ parser = &rowParser{}

func (p *rowParser) decodeItem(i interface{}, dec *json.Decoder) error {
	row := i.(*driver.Row)
	target := struct {
		*driver.Row
		Value json.RawMessage `json:"value"`
		Doc   json.RawMessage `json:"doc"`
	}{
		Row: row,
	}
	if err := dec.Decode(&target); err != nil {
		return err
	}
	if len(target.Value) > 0 {
		row.Value = bytes.NewReader(target.Value)
	}
	if len(target.Doc) > 0 {
		row.Doc = bytes.NewReader(target.Doc)
	}
	return nil
}

func newRows(ctx context.Context, in io.ReadCloser) driver.Rows {
	meta := &rowsMeta{}
	return &rows{
		iter: newIter(ctx, meta, "rows", in, &rowParser{}),
		meta: meta,
	}
}

type findParser struct {
	rowsMetaParser
}

var _ parser = &findParser{}

func (p *findParser) decodeItem(i interface{}, dec *json.Decoder) error {
	var doc json.RawMessage
	if err := dec.Decode(&doc); err != nil {
		return err
	}
	row := i.(*driver.Row)
	row.Doc = bytes.NewReader(doc)
	return nil
}

func newFindRows(ctx context.Context, in io.ReadCloser) driver.Rows {
	meta := &rowsMeta{}
	return &rows{
		iter: newIter(ctx, meta, "docs", in, &findParser{}),
		meta: meta,
	}
}

type bulkParser struct {
	rowsMetaParser
}

var _ parser = &bulkParser{}

func (p *bulkParser) decodeItem(i interface{}, dec *json.Decoder) error {
	row := i.(*driver.Row)
	var result bulkResult
	if err := dec.Decode(&result); err != nil {
		return err
	}
	row.ID = result.ID
	row.Doc = bytes.NewReader(result.Docs[0].Doc)
	row.Error = nil
	if err := result.Docs[0].Error; err != nil {
		row.Error = err
	}
	return nil
}

func newBulkGetRows(ctx context.Context, in io.ReadCloser) driver.Rows {
	meta := &rowsMeta{}
	return &rows{
		iter: newIter(ctx, meta, "results", in, &bulkParser{}),
		meta: meta,
	}
}

func (r *rows) Offset() int64 {
	if r.meta == nil {
		return 0
	}
	return r.meta.offset
}

func (r *rows) TotalRows() int64 {
	if r.meta == nil {
		return 0
	}
	return r.meta.totalRows
}

func (r *rows) Warning() string {
	if r.meta == nil {
		return ""
	}
	return r.meta.warning
}

func (r *rows) Bookmark() string {
	if r.meta == nil {
		return ""
	}
	return r.meta.bookmark
}

func (r *rows) UpdateSeq() string {
	if r.meta == nil {
		return ""
	}
	return string(r.meta.updateSeq)
}

func (r *rows) Next(row *driver.Row) error {
	row.Error = nil
	return r.iter.next(row)
}

// parseMeta parses result metadata
func (r *rowsMeta) parseMeta(key string, dec *json.Decoder) error {
	switch key {
	case "update_seq":
		return dec.Decode(&r.updateSeq)
	case "offset":
		return dec.Decode(&r.offset)
	case "total_rows":
		return dec.Decode(&r.totalRows)
	case "warning":
		return dec.Decode(&r.warning)
	case "bookmark":
		return dec.Decode(&r.bookmark)
	default:
		// Just consume the value, since we don't know what it means.
		var discard json.RawMessage
		return dec.Decode(&discard)
	}
}

func newMultiQueriesRows(ctx context.Context, in io.ReadCloser) driver.Rows {
	return &multiQueriesRows{
		ctx: ctx,
		r:   in,
	}
}

type multiQueriesRows struct {
	*rows
	ctx        context.Context
	r          io.ReadCloser
	dec        *json.Decoder
	queryIndex int
	closed     int32

	// legacy indicates this is an old-style iterator, and won't have more than
	// one resultset.
	legacy int32
}

func (r *multiQueriesRows) Next(row *driver.Row) error {
	if atomic.LoadInt32(&r.closed) == 1 {
		return io.EOF
	}
	if r.rows != nil && atomic.LoadInt32(&r.rows.closed) == 1 {
		if err := r.nextQuery(); err != nil {
			return err
		}
	}
	if r.dec == nil {
		if err := r.begin(); err != nil {
			return err
		}
	}
	if err := r.rows.Next(row); err != nil {
		if err == io.EOF && atomic.LoadInt32(&r.legacy) == 0 {
			return driver.EOQ
		}
		return err
	}
	return nil
}

func (r *multiQueriesRows) begin() error {
	r.dec = json.NewDecoder(r.r)
	// consume the first '{'
	if err := consumeDelim(r.dec, json.Delim('{')); err != nil {
		return err
	}
	key, err := nextKey(r.dec)
	if err != nil {
		return err
	}
	if key != "results" {
		// These indicate the server does not support multiple queries; probably
		// an old version.  Fall back to the standard iterator.
		atomic.StoreInt32(&r.legacy, 1)
		keyJSON, _ := json.Marshal(key)
		var in io.ReadCloser = struct {
			io.Reader
			io.Closer
		}{
			Reader: io.MultiReader(
				strings.NewReader("{"),
				bytes.NewReader(keyJSON),
				r.dec.Buffered(),
				r.r),
			Closer: r.r,
		}
		r.rows = newRows(r.ctx, in).(*rows)
		r.rows.body = nil
		r.rows.dec = json.NewDecoder(in)
		return r.rows.begin()
	}
	// consume the opening '['
	if err := consumeDelim(r.dec, json.Delim('[')); err != nil {
		return err
	}
	r.rows = newRows(r.ctx, r.r).(*rows)
	r.rows.body = nil
	r.rows.iter.dec = r.dec
	return r.rows.iter.begin()
}

func (r *multiQueriesRows) nextQuery() error {
	if atomic.LoadInt32(&r.legacy) == 1 {
		if err := r.Close(); err != nil {
			return err
		}
		return io.EOF
	}
	rows := newRows(r.ctx, r.r).(*rows)
	rows.iter.dec = r.dec
	if err := rows.iter.begin(); err != nil {
		var ud unexpectedDelim
		if errors.As(err, &ud); ud == unexpectedDelim(']') {
			if err := r.Close(); err != nil {
				return err
			}
			return io.EOF
		}
		return err
	}
	r.queryIndex++
	r.rows = rows
	r.rows.body = nil
	return nil
}

func (r *multiQueriesRows) Close() error {
	if atomic.AddInt32(&r.closed, 1) > 1 {
		return nil
	}
	r.dec = nil
	if r.rows != nil {
		defer r.rows.Close() // nolint:errcheck
	}
	defer r.r.Close() // nolint:errcheck
	if _, err := io.ReadAll(r.r); err != nil {
		return err
	}
	if err := r.r.Close(); err != nil {
		return err
	}
	if r.rows == nil {
		return nil
	}
	return r.rows.Close()
}

func (r *multiQueriesRows) QueryIndex() int {
	return r.queryIndex
}
