package geocode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CacheEntry struct {
	Query     string    `json:"query"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Found     bool      `json:"found"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Cache struct {
	Entries map[string]CacheEntry `json:"entries"`
}

func LoadCache(path string) (*Cache, error) {
	if strings.TrimSpace(path) == "" {
		return &Cache{Entries: map[string]CacheEntry{}}, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Cache{Entries: map[string]CacheEntry{}}, nil
		}
		return nil, err
	}
	var cache Cache
	if err := json.Unmarshal(payload, &cache); err != nil {
		return nil, err
	}
	if cache.Entries == nil {
		cache.Entries = map[string]CacheEntry{}
	}
	return &cache, nil
}

func SaveCache(path string, cache *Cache) error {
	if cache == nil {
		return nil
	}
	if strings.TrimSpace(path) == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func (c *Cache) Get(query string) (CacheEntry, bool) {
	if c == nil {
		return CacheEntry{}, false
	}
	key := normalizeQuery(query)
	entry, ok := c.Entries[key]
	return entry, ok
}

func (c *Cache) Set(query string, entry CacheEntry) {
	if c == nil {
		return
	}
	if c.Entries == nil {
		c.Entries = map[string]CacheEntry{}
	}
	key := normalizeQuery(query)
	entry.Query = query
	c.Entries[key] = entry
}

func normalizeQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}
