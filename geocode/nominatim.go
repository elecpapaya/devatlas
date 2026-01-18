package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultNominatimURL = "https://nominatim.openstreetmap.org"

type Nominatim struct {
	baseURL     string
	httpClient  *http.Client
	userAgent   string
	minInterval time.Duration
	mu          sync.Mutex
	lastRequest time.Time
}

type NominatimOption func(*Nominatim)

func WithBaseURL(baseURL string) NominatimOption {
	return func(n *Nominatim) {
		if strings.TrimSpace(baseURL) != "" {
			n.baseURL = baseURL
		}
	}
}

func WithHTTPClient(client *http.Client) NominatimOption {
	return func(n *Nominatim) {
		if client != nil {
			n.httpClient = client
		}
	}
}

func WithUserAgent(userAgent string) NominatimOption {
	return func(n *Nominatim) {
		n.userAgent = userAgent
	}
}

func WithMinInterval(interval time.Duration) NominatimOption {
	return func(n *Nominatim) {
		n.minInterval = interval
	}
}

func NewNominatim(opts ...NominatimOption) *Nominatim {
	n := &Nominatim{
		baseURL:     DefaultNominatimURL,
		httpClient:  http.DefaultClient,
		userAgent:   "devatlas-geocoder/0.1",
		minInterval: 1 * time.Second,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

func (n *Nominatim) Geocode(ctx context.Context, query string) (Result, error) {
	if strings.TrimSpace(query) == "" {
		return Result{Found: false}, nil
	}
	if n == nil {
		return Result{}, errors.New("geocode: nominatim is nil")
	}
	if n.httpClient == nil {
		n.httpClient = http.DefaultClient
	}

	if err := n.waitRateLimit(ctx); err != nil {
		return Result{}, err
	}

	endpoint := strings.TrimRight(n.baseURL, "/") + "/search"
	params := url.Values{}
	params.Set("format", "json")
	params.Set("limit", "1")
	params.Set("q", query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(n.userAgent) != "" {
		req.Header.Set("User-Agent", n.userAgent)
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("geocode: status %d", resp.StatusCode)
	}

	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return Result{}, err
	}
	if len(results) == 0 {
		return Result{Found: false}, nil
	}
	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return Result{}, err
	}
	lng, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return Result{}, err
	}
	return Result{Lat: lat, Lng: lng, Found: true}, nil
}

func (n *Nominatim) waitRateLimit(ctx context.Context) error {
	if n.minInterval <= 0 {
		return nil
	}
	n.mu.Lock()
	now := time.Now()
	next := n.lastRequest.Add(n.minInterval)
	if next.Before(now) || next.Equal(now) {
		n.lastRequest = now
		n.mu.Unlock()
		return nil
	}
	n.lastRequest = next
	n.mu.Unlock()
	delay := time.Until(next)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
