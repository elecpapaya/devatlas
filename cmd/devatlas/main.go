package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devatlas/aggregate"
	"devatlas/geocode"
	"devatlas/mapper"
	"devatlas/saramin"
)

const (
	outputPath          = "data/region_counts.json"
	latestCompaniesPath = "data/latest_companies.json"
	missingRegionPath   = "data/region_missing.jsonl"
	defaultWindow       = 24 * time.Hour
	defaultCurrentDays  = 21
	defaultMinInterval  = 200 * time.Millisecond
	defaultRetryBase    = 500 * time.Millisecond
	defaultRetryMax     = 5 * time.Second
	defaultRetryMaxTry  = 3
	geocodeCachePath    = "data/geocode_cache.json"
)

type runConfig struct {
	accessKey   string
	jobCodes    []string
	updatedMin  int64
	updatedMax  int64
	minInterval time.Duration
	retry       saramin.RetryConfig
	currentDays int
}

type runResult struct {
	pages          int
	jobs           int
	missingRegions int
	elapsed        time.Duration
}

type regionCountsMeta struct {
	RunAt          time.Time `json:"run_at"`
	WindowStart    time.Time `json:"window_start"`
	WindowEnd      time.Time `json:"window_end"`
	MissingRegions int       `json:"missing_regions"`
}

type regionCountsOutput struct {
	Meta    regionCountsMeta        `json:"meta"`
	Regions []aggregate.RegionCount `json:"regions"`
}

type regionIssue struct {
	JobID         string    `json:"job_id,omitempty"`
	Company       string    `json:"company,omitempty"`
	Title         string    `json:"title,omitempty"`
	LocationNames []string  `json:"location_names,omitempty"`
	LocationCodes []string  `json:"location_codes,omitempty"`
	ObservedAt    time.Time `json:"observed_at"`
}

type latestCompaniesMeta struct {
	RunAt       time.Time `json:"run_at"`
	RegionLevel string    `json:"region_level"`
}

type latestCompany struct {
	Name   string  `json:"name"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
	Region string  `json:"region"`
	URL    string  `json:"url"`
	AsOf   string  `json:"asof"`
}

type latestCompaniesOutput struct {
	Meta      latestCompaniesMeta `json:"meta"`
	Companies []latestCompany     `json:"companies"`
}

type geocodeResolver struct {
	resolver *geocode.Resolver
	cache    *geocode.Cache
}

func main() {
	var (
		accessKey     = flag.String("access-key", "", "Saramin access key (or SARAMIN_ACCESS_KEY)")
		jobCd         = flag.String("job-cd", "", "Comma-separated job codes")
		updatedMin    = flag.Int64("updated-min", 0, "Updated min (unix seconds)")
		updatedMax    = flag.Int64("updated-max", 0, "Updated max (unix seconds)")
		currentDays   = flag.Int("current-days", defaultCurrentDays, "Current hiring window in days")
		minIntervalMs = flag.Int("min-interval-ms", int(defaultMinInterval.Milliseconds()), "Minimum interval between API calls in ms")
		retryAttempts = flag.Int("retry-attempts", defaultRetryMaxTry, "Max retry attempts for API calls")
		retryBaseMs   = flag.Int("retry-base-ms", int(defaultRetryBase.Milliseconds()), "Retry base delay in ms")
		retryMaxMs    = flag.Int("retry-max-ms", int(defaultRetryMax.Milliseconds()), "Retry max delay in ms")
	)
	flag.Parse()

	key := strings.TrimSpace(*accessKey)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("SARAMIN_ACCESS_KEY"))
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "missing access key (set -access-key or SARAMIN_ACCESS_KEY)")
		os.Exit(2)
	}

	jobCodes := splitCSV(*jobCd)
	if len(jobCodes) == 0 {
		jobCodes = append([]string(nil), defaultJobCodes...)
	}

	cfg := runConfig{
		accessKey:   key,
		jobCodes:    jobCodes,
		updatedMin:  *updatedMin,
		updatedMax:  *updatedMax,
		minInterval: time.Duration(max(0, *minIntervalMs)) * time.Millisecond,
		retry: saramin.RetryConfig{
			MaxAttempts: max(1, *retryAttempts),
			BaseDelay:   time.Duration(max(0, *retryBaseMs)) * time.Millisecond,
			MaxDelay:    time.Duration(max(0, *retryMaxMs)) * time.Millisecond,
		},
		currentDays: max(1, *currentDays),
	}

	applyRetryDefaults(&cfg)

	result, err := runOnce(context.Background(), cfg, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("pages=%d jobs=%d missing_regions=%d elapsed=%s\n", result.pages, result.jobs, result.missingRegions, result.elapsed.Round(time.Millisecond))
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

var defaultJobCodes = []string{
	"80",
	"82",
	"83",
	"84",
	"85",
	"86",
	"87",
	"90",
	"92",
	"95",
	"99",
	"100",
	"101",
	"103",
	"104",
	"106",
	"107",
	"108",
	"109",
	"111",
	"113",
	"116",
	"123",
	"124",
	"127",
	"128",
	"131",
	"132",
	"133",
	"135",
	"136",
	"139",
	"142",
	"145",
	"146",
	"148",
	"150",
	"156",
	"160",
	"161",
	"162",
	"164",
	"171",
	"172",
	"180",
	"181",
	"195",
	"234",
	"320",
	"2229",
	"2232",
	"2239",
	"2246",
	"2248",
	"2249",
}

func writeRegionCounts(path string, meta regionCountsMeta, stats []aggregate.RegionCount) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(regionCountsOutput{
		Meta:    meta,
		Regions: stats,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func appendRegionIssues(path string, issues []regionIssue) error {
	if len(issues) == 0 {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, issue := range issues {
		payload, err := json.Marshal(issue)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(payload, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func writeLatestCompanies(path string, meta latestCompaniesMeta, companies []aggregate.CompanyRecord) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	out := latestCompaniesOutput{
		Meta:      meta,
		Companies: make([]latestCompany, 0, len(companies)),
	}
	for _, company := range companies {
		out.Companies = append(out.Companies, latestCompany{
			Name:   company.Name,
			Lat:    company.Lat,
			Lng:    company.Lng,
			Region: company.Region,
			URL:    company.URL,
			AsOf:   company.LastSeen.Format("2006-01-02"),
		})
	}
	payload, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func initGeocodeResolver() (geocodeResolver, error) {
	cache, err := geocode.LoadCache(geocodeCachePath)
	if err != nil {
		return geocodeResolver{}, err
	}
	geocoder := geocode.NewNominatim()
	return geocodeResolver{
		resolver: geocode.NewResolver(geocoder, cache),
		cache:    cache,
	}, nil
}

func buildGeoQuery(locationNames []string) string {
	for _, name := range locationNames {
		candidate := strings.TrimSpace(name)
		if candidate == "" {
			continue
		}
		if containsRemoteKeyword(candidate) {
			continue
		}
		candidate = strings.NewReplacer(">", " ", "/", " ", ",", " ").Replace(candidate)
		candidate = strings.Join(strings.Fields(candidate), " ")
		if candidate == "" {
			continue
		}
		return candidate
	}
	return ""
}

func containsRemoteKeyword(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "전국") ||
		strings.Contains(lower, "재택") ||
		strings.Contains(lower, "원격") ||
		strings.Contains(lower, "리모트") ||
		strings.Contains(lower, "remote") ||
		strings.Contains(lower, "해외")
}

func runOnce(ctx context.Context, cfg runConfig, now time.Time) (runResult, error) {
	started := time.Now()
	windowStart, windowEnd, err := resolveWindow(cfg, now)
	if err != nil {
		return runResult{}, err
	}

	baseParams := saramin.JobSearchParams{
		JobCd: cfg.jobCodes,
		Sr:    []string{"directhire"},
		Count: saramin.DefaultPageSize,
		Sort:  "ud",
	}
	client := saramin.NewClient(
		cfg.accessKey,
		saramin.WithMinInterval(cfg.minInterval),
		saramin.WithRetryConfig(cfg.retry),
	)
	observedAt := now

	geo, err := initGeocodeResolver()
	if err != nil {
		return runResult{}, err
	}
	if geo.cache != nil {
		defer func() {
			_ = geocode.SaveCache(geocodeCachePath, geo.cache)
		}()
	}

	regionAgg := aggregate.NewRegionAggregator()
	companyAgg := aggregate.NewCompanyAggregator()
	missingRegionIDs := map[string]struct{}{}
	issues := make([]regionIssue, 0)
	pages, jobs, missing, err := collectWindow(ctx, client, baseParams, windowStart, windowEnd, regionAgg, companyAgg, geo.resolver, observedAt, missingRegionIDs, &issues)
	if err != nil {
		return runResult{}, err
	}

	missingCount := missing

	stats := regionAgg.Results()
	meta := regionCountsMeta{
		RunAt:          now,
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		MissingRegions: missingCount,
	}
	if err := writeRegionCounts(outputPath, meta, stats); err != nil {
		return runResult{}, err
	}

	activeCompanies := companyAgg.ActiveCompanies(now.AddDate(0, 0, -cfg.currentDays))
	if err := writeLatestCompanies(latestCompaniesPath, latestCompaniesMeta{
		RunAt:       now,
		RegionLevel: "sido",
	}, activeCompanies); err != nil {
		return runResult{}, err
	}
	if len(issues) > 0 {
		if err := appendRegionIssues(missingRegionPath, issues); err != nil {
			return runResult{}, err
		}
	}

	return runResult{
		pages:          pages,
		jobs:           jobs,
		missingRegions: missingCount,
		elapsed:        time.Since(started),
	}, nil
}

func resolveWindow(cfg runConfig, now time.Time) (time.Time, time.Time, error) {
	var start, end time.Time
	if cfg.updatedMin > 0 {
		start = time.Unix(cfg.updatedMin, 0)
	}
	if cfg.updatedMax > 0 {
		end = time.Unix(cfg.updatedMax, 0)
	}
	if start.IsZero() && end.IsZero() {
		end = now
		start = now.Add(-defaultWindow)
	} else if start.IsZero() && !end.IsZero() {
		start = end.Add(-defaultWindow)
	} else if !start.IsZero() && end.IsZero() {
		end = now
	}
	if !start.Before(end) {
		start = end.Add(-defaultWindow)
	}
	return start, end, nil
}

func collectWindow(
	ctx context.Context,
	client *saramin.Client,
	baseParams saramin.JobSearchParams,
	windowStart time.Time,
	windowEnd time.Time,
	regionAgg *aggregate.RegionAggregator,
	companyAgg *aggregate.CompanyAggregator,
	geo *geocode.Resolver,
	observedAt time.Time,
	missingIDs map[string]struct{},
	issues *[]regionIssue,
) (int, int, int, error) {
	if windowStart.IsZero() || windowEnd.IsZero() {
		return 0, 0, 0, errors.New("invalid window range")
	}
	if !windowStart.Before(windowEnd) {
		return 0, 0, 0, nil
	}

	params := baseParams
	params.UpdatedMin = windowStart
	params.UpdatedMax = windowEnd

	var pages int
	var jobs int
	var missing int
	err := client.JobSearchPages(ctx, params, func(resp *saramin.JobSearchResponse) error {
		pages++
		for _, job := range resp.Jobs.Job {
			normalized := mapper.NormalizeSaraminJob(job, observedAt)
			if geo != nil {
				query := buildGeoQuery(normalized.LocationNames)
				if query != "" {
					result, _, err := geo.Resolve(ctx, query)
					if err != nil {
						return err
					}
					if result.Found {
						normalized.Latitude = result.Lat
						normalized.Longitude = result.Lng
					}
				}
			}
			regionAgg.Add(normalized)
			if companyAgg != nil {
				companyAgg.Add(normalized)
			}
			if normalized.Region == "" {
				if job.ID != "" && missingIDs != nil {
					if _, exists := missingIDs[job.ID]; exists {
						continue
					}
					missingIDs[job.ID] = struct{}{}
				}
				missing++
				if issues != nil {
					*issues = append(*issues, regionIssue{
						JobID:         job.ID,
						Company:       normalized.CompanyName,
						Title:         normalized.Title,
						LocationNames: normalized.LocationNames,
						LocationCodes: normalized.LocationCodes,
						ObservedAt:    observedAt,
					})
				}
			}
		}
		jobs += len(resp.Jobs.Job)
		return nil
	})
	if err != nil {
		return 0, 0, 0, err
	}
	return pages, jobs, missing, nil
}

func applyRetryDefaults(cfg *runConfig) {
	if cfg == nil {
		return
	}
	if cfg.minInterval <= 0 {
		cfg.minInterval = defaultMinInterval
	}
	if cfg.retry.MaxAttempts < 1 {
		cfg.retry.MaxAttempts = defaultRetryMaxTry
	}
	if cfg.retry.BaseDelay <= 0 {
		cfg.retry.BaseDelay = defaultRetryBase
	}
	if cfg.retry.MaxDelay <= 0 {
		cfg.retry.MaxDelay = defaultRetryMax
	}
	cfg.retry.StatusCodes = map[int]struct{}{
		429: {},
		500: {},
		502: {},
		503: {},
		504: {},
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
