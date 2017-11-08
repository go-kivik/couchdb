package couchdb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
	"github.com/go-kivik/couchdb/chttp"
)

type schedulerDoc struct {
	Database      string    `json:"database"`
	DocID         string    `json:"doc_id"`
	ReplicationID string    `json:"id"`
	Source        string    `json:"source"`
	Target        string    `json:"target"`
	StartTime     time.Time `json:"start_time"`
	LastUpdated   time.Time `json:"last_updated"`
	State         string    `json:"state"`
	Info          repInfo   `json:"info"`
}

type repInfo struct {
	Error            error
	DocsRead         int64 `json:"docs_read"`
	DocsWritten      int64 `json:"docs_written"`
	DocWriteFailures int64 `json:"doc_write_failures"`
	Pending          int64 `json:"changes_pending"`
}

func (i *repInfo) UnmarshalJSON(data []byte) error {
	switch {
	case string(data) == "null":
		return nil
	case data[0] == '{':
		type repInfoClone repInfo
		var x repInfoClone
		if err := json.Unmarshal(data, &x); err != nil {
			return err
		}
		*i = repInfo(x)
	default:
		var e replicationError
		if err := json.Unmarshal(data, &e); err != nil {
			return err
		}
		i.Error = &e
	}
	return nil
}

type schedulerReplication struct {
	docID         string
	database      string
	replicationID string
	source        string
	target        string
	startTime     time.Time
	endTime       time.Time
	state         string
	err           error

	*db
}

var _ driver.Replication = &schedulerReplication{}

func (c *client) newSchedulerReplication(doc *schedulerDoc) *schedulerReplication {
	rep := &schedulerReplication{
		docID:         doc.DocID,
		database:      doc.Database,
		replicationID: doc.ReplicationID,
		source:        doc.Source,
		target:        doc.Target,
		startTime:     doc.StartTime,
		state:         doc.State,
		db: &db{
			client: c,
			dbName: doc.Database,
		},
	}
	if doc.Info.Error != nil {
		rep.err = doc.Info.Error
	}
	switch doc.State {
	case "failed", "completed":
		rep.endTime = doc.LastUpdated
	}
	return rep
}

func (r *schedulerReplication) StartTime() time.Time  { return r.startTime }
func (r *schedulerReplication) EndTime() time.Time    { return r.endTime }
func (r *schedulerReplication) Err() error            { return r.err }
func (r *schedulerReplication) ReplicationID() string { return r.replicationID }
func (r *schedulerReplication) Source() string        { return r.source }
func (r *schedulerReplication) Target() string        { return r.target }
func (r *schedulerReplication) State() string         { return r.state }

func (r *schedulerReplication) Update(ctx context.Context, rep *driver.ReplicationInfo) error {
	doc, err := r.getSchedulerDoc(ctx)
	if err != nil {
		return err
	}
	if doc.Info.Error == nil {
		r.err = doc.Info.Error
	}
	rep.DocWriteFailures = doc.Info.DocWriteFailures
	rep.DocsRead = doc.Info.DocsRead
	rep.DocsWritten = doc.Info.DocsWritten
	return nil
}

func (r *schedulerReplication) Delete(ctx context.Context) error {
	return nil
}

func (r *schedulerReplication) getSchedulerDoc(ctx context.Context) (*schedulerDoc, error) {
	path := fmt.Sprintf("/_scheduler/docs/%s/%s", r.database, chttp.EncodeDocID(r.docID))
	var doc schedulerDoc
	_, err := r.db.Client.DoJSON(ctx, kivik.MethodGet, path, nil, &doc)
	return &doc, err
}

// errSchedulerNotImplemented is used internally only, and signals that a
// fallback should occur to the legacy replicator database.
var errSchedulerNotImplemented = errors.Status(kivik.StatusNotImplemented, "_scheduler interface not implemented")

func (c *client) getReplicationsFromScheduler(ctx context.Context, options map[string]interface{}) ([]driver.Replication, error) {
	params, err := optionsToParams(options)
	if err != nil {
		return nil, err
	}
	var result struct {
		Docs []schedulerDoc `json:"docs"`
	}
	path := "/_scheduler/docs"
	if params != nil {
		path = path + "?" + params.Encode()
	}
	if _, err = c.DoJSON(ctx, kivik.MethodGet, path, nil, &result); err != nil {
		if code := kivik.StatusCode(err); code == kivik.StatusNotFound || code == kivik.StatusBadRequest {
			return nil, errSchedulerNotImplemented
		}
		return nil, err
	}
	reps := make([]driver.Replication, 0, len(result.Docs))
	for _, row := range result.Docs {
		rep := c.newSchedulerReplication(&row)
		reps = append(reps, rep)
	}
	return reps, nil
}
