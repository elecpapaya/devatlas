package saramin

import (
	"encoding/json"
)

type JobSearchResponse struct {
	Jobs    JobSearchJobs `json:"jobs"`
	Code    *int          `json:"code,omitempty"`
	Message *string       `json:"message,omitempty"`
}

type JobSearchJobs struct {
	Count int    `json:"count"`
	Start int    `json:"start"`
	Total string `json:"total"`
	Job   []Job  `json:"job"`
}

type Job struct {
	URL                   string         `json:"url"`
	Active                StringOrNumber `json:"active"`
	Company               Company        `json:"company"`
	Position              Position       `json:"position"`
	Keyword               string         `json:"keyword"`
	Salary                CodeName       `json:"salary"`
	ID                    string         `json:"id"`
	PostingTimestamp      StringOrNumber `json:"posting-timestamp"`
	PostingDate           string         `json:"posting-date"`
	ModificationTimestamp StringOrNumber `json:"modification-timestamp"`
	OpeningTimestamp      StringOrNumber `json:"opening-timestamp"`
	ExpirationTimestamp   StringOrNumber `json:"expiration-timestamp"`
	ExpirationDate        string         `json:"expiration-date"`
	CloseType             CodeName       `json:"close-type"`
	ReadCnt               StringOrNumber `json:"read-cnt"`
	ApplyCnt              StringOrNumber `json:"apply-cnt"`
}

type Company struct {
	Detail CompanyDetail `json:"detail"`
}

type CompanyDetail struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

type Position struct {
	Title                  string     `json:"title"`
	Industry               CodeName   `json:"industry"`
	Location               CodeName   `json:"location"`
	JobType                CodeName   `json:"job-type"`
	JobMidCode             CodeName   `json:"job-mid-code"`
	JobCode                CodeName   `json:"job-code"`
	ExperienceLevel        Experience `json:"experience-level"`
	RequiredEducationLevel CodeName   `json:"required-education-level"`
	IndustryKeywordCode    string     `json:"industry-keyword-code"`
	JobCodeKeywordCode     string     `json:"job-code-keyword-code"`
}

type CodeName struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Experience struct {
	Code StringOrNumber `json:"code"`
	Min  StringOrNumber `json:"min"`
	Max  StringOrNumber `json:"max"`
	Name string         `json:"name"`
}

type APIErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	Code       int
	Message    string
	StatusCode int
}

func (e *APIError) Error() string {
	if e == nil {
		return "saramin: api error"
	}
	if e.Message == "" {
		return "saramin: api error"
	}
	return "saramin: " + e.Message
}

type StringOrNumber string

func (s *StringOrNumber) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*s = StringOrNumber(v)
		return nil
	}
	var v json.Number
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*s = StringOrNumber(v.String())
	return nil
}
