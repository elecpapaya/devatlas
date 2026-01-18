package saramin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultBaseURL = "https://oapi.saramin.co.kr"

var defaultFields = []string{"posting-date", "expiration-date", "count"}

type Client struct {
	baseURL     string
	accessKey   string
	httpClient  *http.Client
	userAgent   string
	minInterval time.Duration
	retry       RetryConfig
	mu          sync.Mutex
	lastRequest time.Time
}

type Option func(*Client)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	StatusCodes map[int]struct{}
}

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) != "" {
			c.baseURL = baseURL
		}
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

func WithMinInterval(interval time.Duration) Option {
	return func(c *Client) {
		c.minInterval = interval
	}
}

func WithRetryConfig(cfg RetryConfig) Option {
	return func(c *Client) {
		c.retry = cfg
	}
}

func NewClient(accessKey string, opts ...Option) *Client {
	client := &Client{
		baseURL:     DefaultBaseURL,
		accessKey:   accessKey,
		httpClient:  http.DefaultClient,
		userAgent:   "devatlas-saramin-client/0.1",
		minInterval: 200 * time.Millisecond,
		retry: RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   500 * time.Millisecond,
			MaxDelay:    5 * time.Second,
			StatusCodes: map[int]struct{}{
				http.StatusTooManyRequests:     {},
				http.StatusInternalServerError: {},
				http.StatusBadGateway:          {},
				http.StatusServiceUnavailable:  {},
				http.StatusGatewayTimeout:      {},
			},
		},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

type JobSearchParams struct {
	Keywords     []string
	JobCd        []string
	JobMidCd     []string
	LocCd        []string
	Sr           []string
	Fields       []string
	UpdatedMin   time.Time
	UpdatedMax   time.Time
	PublishedMin time.Time
	PublishedMax time.Time
	Start        int
	Count        int
	Sort         string
}

func (p JobSearchParams) Encode(accessKey string) (url.Values, error) {
	if strings.TrimSpace(accessKey) == "" {
		return nil, errors.New("saramin: access key is required")
	}

	values := url.Values{}
	values.Set("access-key", accessKey)

	if len(p.Keywords) > 0 {
		values.Set("keywords", strings.Join(p.Keywords, ","))
	}
	if len(p.JobCd) > 0 {
		values.Set("job_cd", strings.Join(p.JobCd, ","))
	}
	if len(p.JobMidCd) > 0 {
		values.Set("job_mid_cd", strings.Join(p.JobMidCd, ","))
	}
	if len(p.LocCd) > 0 {
		values.Set("loc_cd", strings.Join(p.LocCd, ","))
	}
	if len(p.Sr) > 0 {
		values.Set("sr", strings.Join(p.Sr, ","))
	}
	if len(p.Fields) > 0 {
		values.Set("fields", strings.Join(p.Fields, ","))
	}
	if !p.UpdatedMin.IsZero() {
		values.Set("updated_min", formatTime(p.UpdatedMin))
	}
	if !p.UpdatedMax.IsZero() {
		values.Set("updated_max", formatTime(p.UpdatedMax))
	}
	if !p.PublishedMin.IsZero() {
		values.Set("published_min", formatTime(p.PublishedMin))
	}
	if !p.PublishedMax.IsZero() {
		values.Set("published_max", formatTime(p.PublishedMax))
	}
	if p.Start > 0 {
		values.Set("start", strconv.Itoa(p.Start))
	}
	if p.Count > 0 {
		values.Set("count", strconv.Itoa(p.Count))
	}
	if strings.TrimSpace(p.Sort) != "" {
		values.Set("sort", p.Sort)
	}

	return values, nil
}

func (c *Client) JobSearch(ctx context.Context, params JobSearchParams) (*JobSearchResponse, error) {
	if c == nil {
		return nil, errors.New("saramin: client is nil")
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	if len(params.Fields) == 0 {
		params.Fields = append([]string(nil), defaultFields...)
	}

	values, err := params.Encode(c.accessKey)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/job-search"
	query := values.Encode()

	maxAttempts := c.retry.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.URL.RawQuery = query
		req.Header.Set("Accept", "application/json")
		if strings.TrimSpace(c.userAgent) != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}

		statusCode, body, err := c.doRequest(ctx, req)
		if err == nil && statusCode == http.StatusOK {
			var out JobSearchResponse
			if err := json.Unmarshal(body, &out); err != nil {
				return nil, err
			}
			if out.Code != nil && *out.Code != 0 {
				message := ""
				if out.Message != nil {
					message = *out.Message
				}
				return nil, &APIError{Code: *out.Code, Message: message, StatusCode: statusCode}
			}
			return &out, nil
		}

		lastErr = err
		if !c.shouldRetry(statusCode, err) || attempt == maxAttempts {
			if err != nil {
				return nil, err
			}
			return nil, decodeAPIError(statusCode, body)
		}

		if err := sleepWithContext(ctx, c.retryDelay(attempt)); err != nil {
			return nil, err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("saramin: request failed")
}

func decodeAPIError(statusCode int, body []byte) error {
	var apiErr APIErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Code != 0 {
		return &APIError{
			Code:       apiErr.Code,
			Message:    apiErr.Message,
			StatusCode: statusCode,
		}
	}
	return fmt.Errorf("saramin: http status %d", statusCode)
}

func formatTime(value time.Time) string {
	return strconv.FormatInt(value.Unix(), 10)
}

func (c *Client) doRequest(ctx context.Context, req *http.Request) (int, []byte, error) {
	if err := c.waitRateLimit(ctx); err != nil {
		return 0, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

func (c *Client) waitRateLimit(ctx context.Context) error {
	if c.minInterval <= 0 {
		return nil
	}
	c.mu.Lock()
	now := time.Now()
	next := c.lastRequest.Add(c.minInterval)
	if next.Before(now) || next.Equal(now) {
		c.lastRequest = now
		c.mu.Unlock()
		return nil
	}
	c.lastRequest = next
	c.mu.Unlock()

	return sleepWithContext(ctx, time.Until(next))
}

func (c *Client) shouldRetry(statusCode int, err error) bool {
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		return true
	}
	_, ok := c.retry.StatusCodes[statusCode]
	return ok
}

func (c *Client) retryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return c.retry.BaseDelay
	}
	delay := c.retry.BaseDelay << (attempt - 1)
	if delay > c.retry.MaxDelay {
		delay = c.retry.MaxDelay
	}
	if delay <= 0 {
		return 200 * time.Millisecond
	}
	return delay
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
