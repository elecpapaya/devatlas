package runlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Status string

const (
	StatusStarted   Status = "started"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type RunRecord struct {
	ID          string            `json:"id"`
	RunAt       time.Time         `json:"run_at"`
	WindowStart time.Time         `json:"window_start"`
	WindowEnd   time.Time         `json:"window_end"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt time.Time         `json:"completed_at,omitempty"`
	Status      Status            `json:"status"`
	Error       string            `json:"error,omitempty"`
	Metrics     map[string]int64  `json:"metrics,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

type Recorder struct {
	dir string
	now func() time.Time
}

func NewRecorder(dir string) *Recorder {
	return &Recorder{
		dir: dir,
		now: time.Now,
	}
}

func (r *Recorder) Start(runAt, windowStart, windowEnd time.Time) (*RunRecord, error) {
	if r == nil {
		return nil, errors.New("runlog: recorder is nil")
	}
	if r.dir == "" {
		return nil, errors.New("runlog: directory is required")
	}

	if runAt.IsZero() {
		runAt = r.now()
	}
	record := &RunRecord{
		ID:          runAt.Format("20060102T150405Z0700"),
		RunAt:       runAt,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		StartedAt:   r.now(),
		Status:      StatusStarted,
	}
	if err := r.write(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (r *Recorder) Finish(record *RunRecord, runErr error) error {
	if r == nil {
		return errors.New("runlog: recorder is nil")
	}
	if record == nil {
		return errors.New("runlog: record is nil")
	}
	record.CompletedAt = r.now()
	if runErr != nil {
		record.Status = StatusFailed
		record.Error = runErr.Error()
	} else {
		record.Status = StatusCompleted
		record.Error = ""
	}
	return r.write(record)
}

func (r *Recorder) write(record *RunRecord) error {
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	path := filepath.Join(r.dir, fmt.Sprintf("run-%s.json", record.ID))
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}
