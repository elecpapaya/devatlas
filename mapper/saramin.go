package mapper

import (
	"strconv"
	"strings"
	"time"

	"devatlas/model"
	"devatlas/saramin"
)

func NormalizeSaraminJob(job saramin.Job, observedAt time.Time) model.NormalizedJob {
	if observedAt.IsZero() {
		observedAt = time.Now()
	}

	locationCodes := splitCSV(job.Position.Location.Code)
	locationNames := splitCSV(job.Position.Location.Name)

	return model.NormalizedJob{
		Source:        "saramin",
		SourceJobID:   job.ID,
		SourceURL:     job.URL,
		CompanyName:   job.Company.Detail.Name,
		CompanyURL:    job.Company.Detail.Href,
		Title:         job.Position.Title,
		JobMidCode:    job.Position.JobMidCode.Code,
		JobCode:       job.Position.JobCode.Code,
		JobTypeCode:   job.Position.JobType.Code,
		LocationCodes: locationCodes,
		LocationNames: locationNames,
		Region:        extractRegion(locationNames),
		Keywords:      splitCSV(job.Keyword),
		Active:        parseActive(job.Active),
		PostedAt:      parseUnix(job.PostingTimestamp),
		UpdatedAt:     parseUnix(job.ModificationTimestamp),
		ExpiresAt:     parseUnix(job.ExpirationTimestamp),
		ObservedAt:    observedAt,
	}
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

func extractRegion(locationNames []string) string {
	if len(locationNames) == 0 {
		return ""
	}
	for _, name := range locationNames {
		region := extractRegionFromName(name)
		if region != "" {
			return region
		}
	}
	return ""
}

func extractRegionFromName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	candidates := strings.Split(trimmed, ",")
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if idx := strings.Index(candidate, ">"); idx >= 0 {
			candidate = candidate[:idx]
		}
		if idx := strings.Index(candidate, "/"); idx >= 0 {
			candidate = candidate[:idx]
		}
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if parts := strings.Fields(candidate); len(parts) > 0 {
			candidate = parts[0]
		}
		if region := normalizeRegionName(candidate); region != "" {
			return region
		}
	}
	return ""
}

type regionAlias struct {
	prefix    string
	canonical string
}

var regionAliases = []regionAlias{
	{"서울특별시", "서울"},
	{"서울시", "서울"},
	{"서울", "서울"},
	{"부산광역시", "부산"},
	{"부산시", "부산"},
	{"부산", "부산"},
	{"대구광역시", "대구"},
	{"대구시", "대구"},
	{"대구", "대구"},
	{"인천광역시", "인천"},
	{"인천시", "인천"},
	{"인천", "인천"},
	{"광주광역시", "광주"},
	{"광주시", "광주"},
	{"광주", "광주"},
	{"대전광역시", "대전"},
	{"대전시", "대전"},
	{"대전", "대전"},
	{"울산광역시", "울산"},
	{"울산시", "울산"},
	{"울산", "울산"},
	{"세종특별자치시", "세종"},
	{"세종시", "세종"},
	{"세종", "세종"},
	{"경기도", "경기"},
	{"경기", "경기"},
	{"강원특별자치도", "강원"},
	{"강원도", "강원"},
	{"강원", "강원"},
	{"충청북도", "충북"},
	{"충북", "충북"},
	{"충청남도", "충남"},
	{"충남", "충남"},
	{"전라북도", "전북"},
	{"전북", "전북"},
	{"전라남도", "전남"},
	{"전남", "전남"},
	{"경상북도", "경북"},
	{"경북", "경북"},
	{"경상남도", "경남"},
	{"경남", "경남"},
	{"제주특별자치도", "제주"},
	{"제주도", "제주"},
	{"제주", "제주"},
}

func normalizeRegionName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if idx := strings.Index(trimmed, "("); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "전국") || strings.Contains(trimmed, "재택") || strings.Contains(trimmed, "원격") || strings.Contains(trimmed, "해외") {
		return ""
	}
	for _, alias := range regionAliases {
		if strings.HasPrefix(trimmed, alias.prefix) {
			return alias.canonical
		}
	}
	return ""
}

func parseUnix(value saramin.StringOrNumber) time.Time {
	raw := strings.TrimSpace(string(value))
	if raw == "" {
		return time.Time{}
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(seconds, 0)
}

func parseActive(value saramin.StringOrNumber) bool {
	raw := strings.TrimSpace(string(value))
	return raw == "1"
}
