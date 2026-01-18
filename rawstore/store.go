package rawstore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"devatlas/model"
)

type FileStore struct {
	dir         string
	now         func() time.Time
	currentDate string
	file        *os.File
	writer      *bufio.Writer
}

func NewFileStore(dir string) *FileStore {
	return &FileStore{
		dir: dir,
		now: time.Now,
	}
}

func (s *FileStore) Append(job model.RawJob) error {
	if s == nil {
		return fmt.Errorf("rawstore: store is nil")
	}
	if s.dir == "" {
		return fmt.Errorf("rawstore: directory is required")
	}
	if job.FetchedAt.IsZero() {
		job.FetchedAt = s.now()
	}

	dateKey := job.FetchedAt.Format("20060102")
	if err := s.ensureWriter(dateKey); err != nil {
		return err
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	if _, err := s.writer.Write(append(payload, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *FileStore) Close() error {
	if s == nil {
		return nil
	}
	if s.writer != nil {
		if err := s.writer.Flush(); err != nil {
			return err
		}
	}
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			return err
		}
	}
	s.writer = nil
	s.file = nil
	s.currentDate = ""
	return nil
}

func (s *FileStore) ensureWriter(dateKey string) error {
	if s.writer != nil && s.currentDate == dateKey {
		return nil
	}
	if err := s.rotate(dateKey); err != nil {
		return err
	}
	return nil
}

func (s *FileStore) rotate(dateKey string) error {
	if err := s.Close(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	name := fmt.Sprintf("raw-%s.jsonl", dateKey)
	path := filepath.Join(s.dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	s.file = file
	s.writer = bufio.NewWriterSize(file, 64*1024)
	s.currentDate = dateKey
	return nil
}
