package model

import "time"

type NormalizedJob struct {
	Source        string
	SourceJobID   string
	SourceURL     string
	CompanyName   string
	CompanyURL    string
	Title         string
	JobMidCode    string
	JobCode       string
	JobTypeCode   string
	LocationCodes []string
	LocationNames []string
	Region        string
	Keywords      []string
	Active        bool
	Latitude      float64
	Longitude     float64
	PostedAt      time.Time
	UpdatedAt     time.Time
	ExpiresAt     time.Time
	ObservedAt    time.Time
}

type NormalizedCompany struct {
	Name       string
	Region     string
	Latitude   float64
	Longitude  float64
	SourceURL  string
	ObservedAt time.Time
}
