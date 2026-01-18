package aggregate

import (
	"sort"
	"strings"
	"time"

	"devatlas/model"
)

type CompanyRecord struct {
	Name     string
	Region   string
	Lat      float64
	Lng      float64
	URL      string
	LastSeen time.Time
}

type CompanyAggregator struct {
	records map[string]*CompanyRecord
}

func NewCompanyAggregator() *CompanyAggregator {
	return &CompanyAggregator{
		records: map[string]*CompanyRecord{},
	}
}

func (a *CompanyAggregator) Add(job model.NormalizedJob) {
	if a == nil {
		return
	}
	if job.CompanyName == "" {
		return
	}
	region := strings.TrimSpace(job.Region)
	if region == "" {
		return
	}

	lastSeen := pickLatestTime(job.UpdatedAt, job.PostedAt, job.ObservedAt)
	if lastSeen.IsZero() {
		lastSeen = time.Now()
	}

	url := job.CompanyURL
	if url == "" {
		url = job.SourceURL
	}

	key := job.CompanyName + "|" + region
	record, ok := a.records[key]
	if !ok {
		coords := regionCentroids[region]
		if job.Latitude != 0 || job.Longitude != 0 {
			coords = latLng{Lat: job.Latitude, Lng: job.Longitude}
		}
		a.records[key] = &CompanyRecord{
			Name:     job.CompanyName,
			Region:   region,
			Lat:      coords.Lat,
			Lng:      coords.Lng,
			URL:      url,
			LastSeen: lastSeen,
		}
		return
	}

	if lastSeen.After(record.LastSeen) {
		record.LastSeen = lastSeen
		if url != "" {
			record.URL = url
		}
	}
	if record.Lat == 0 && record.Lng == 0 && (job.Latitude != 0 || job.Longitude != 0) {
		record.Lat = job.Latitude
		record.Lng = job.Longitude
	}
}

func (a *CompanyAggregator) ActiveCompanies(cutoff time.Time) []CompanyRecord {
	if a == nil {
		return nil
	}
	out := make([]CompanyRecord, 0, len(a.records))
	for _, record := range a.records {
		if record == nil {
			continue
		}
		if record.LastSeen.IsZero() || record.LastSeen.Before(cutoff) {
			continue
		}
		out = append(out, *record)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Region == out[j].Region {
			return out[i].Name < out[j].Name
		}
		return out[i].Region < out[j].Region
	})
	return out
}

func pickLatestTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.IsZero() {
			continue
		}
		if latest.IsZero() || value.After(latest) {
			latest = value
		}
	}
	return latest
}

type latLng struct {
	Lat float64
	Lng float64
}

var regionCentroids = map[string]latLng{
	"서울": {Lat: 37.5665, Lng: 126.9780},
	"부산": {Lat: 35.1796, Lng: 129.0756},
	"대구": {Lat: 35.8722, Lng: 128.6025},
	"인천": {Lat: 37.4563, Lng: 126.7052},
	"광주": {Lat: 35.1595, Lng: 126.8526},
	"대전": {Lat: 36.3504, Lng: 127.3845},
	"울산": {Lat: 35.5384, Lng: 129.3114},
	"세종": {Lat: 36.4801, Lng: 127.2890},
	"경기": {Lat: 37.4138, Lng: 127.5183},
	"강원": {Lat: 37.8228, Lng: 128.1555},
	"충북": {Lat: 36.6358, Lng: 127.4914},
	"충남": {Lat: 36.5184, Lng: 126.8000},
	"전북": {Lat: 35.8202, Lng: 127.1088},
	"전남": {Lat: 34.8161, Lng: 126.4629},
	"경북": {Lat: 36.5760, Lng: 128.5056},
	"경남": {Lat: 35.4606, Lng: 128.2132},
	"제주": {Lat: 33.4996, Lng: 126.5312},
}
