package aggregate

import (
	"sort"
	"strings"

	"devatlas/model"
)

type RegionCount struct {
	Region       string `json:"region"`
	JobCount     int    `json:"job_count"`
	CompanyCount int    `json:"company_count"`
}

type RegionAggregator struct {
	jobCounts   map[string]int
	jobIDs      map[string]map[string]struct{}
	companySets map[string]map[string]struct{}
}

func NewRegionAggregator() *RegionAggregator {
	return &RegionAggregator{
		jobCounts:   map[string]int{},
		jobIDs:      map[string]map[string]struct{}{},
		companySets: map[string]map[string]struct{}{},
	}
}

func (a *RegionAggregator) Add(job model.NormalizedJob) {
	if a == nil {
		return
	}
	region := strings.TrimSpace(job.Region)
	if region == "" {
		return
	}
	if job.SourceJobID == "" {
		a.jobCounts[region]++
	} else {
		set, ok := a.jobIDs[region]
		if !ok {
			set = map[string]struct{}{}
			a.jobIDs[region] = set
		}
		set[job.SourceJobID] = struct{}{}
	}

	if job.CompanyName == "" {
		return
	}
	set, ok := a.companySets[region]
	if !ok {
		set = map[string]struct{}{}
		a.companySets[region] = set
	}
	set[job.CompanyName] = struct{}{}
}

func (a *RegionAggregator) Results() []RegionCount {
	if a == nil {
		return nil
	}
	regions := make([]string, 0, len(a.jobCounts))
	for region := range a.jobCounts {
		regions = append(regions, region)
	}
	sort.Strings(regions)

	out := make([]RegionCount, 0, len(regions))
	for _, region := range regions {
		jobCount := a.jobCounts[region]
		if set, ok := a.jobIDs[region]; ok {
			jobCount += len(set)
		}
		out = append(out, RegionCount{
			Region:       region,
			JobCount:     jobCount,
			CompanyCount: len(a.companySets[region]),
		})
	}
	return out
}
