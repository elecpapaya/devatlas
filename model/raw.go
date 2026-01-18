package model

import (
	"encoding/json"
	"time"
)

type RawJob struct {
	Source      string
	SourceJobID string
	FetchedAt   time.Time
	Payload     json.RawMessage
}
