package geocode

import (
	"context"
	"strings"
	"time"
)

type Result struct {
	Lat   float64
	Lng   float64
	Found bool
}

type Geocoder interface {
	Geocode(ctx context.Context, query string) (Result, error)
}

type Resolver struct {
	geocoder Geocoder
	cache    *Cache
	now      func() time.Time
}

func NewResolver(geocoder Geocoder, cache *Cache) *Resolver {
	return &Resolver{
		geocoder: geocoder,
		cache:    cache,
		now:      time.Now,
	}
}

func (r *Resolver) Resolve(ctx context.Context, query string) (Result, bool, error) {
	if r == nil || r.geocoder == nil {
		return Result{Found: false}, false, nil
	}
	if strings.TrimSpace(query) == "" {
		return Result{Found: false}, false, nil
	}
	if entry, ok := r.cache.Get(query); ok {
		return Result{Lat: entry.Lat, Lng: entry.Lng, Found: entry.Found}, true, nil
	}
	result, err := r.geocoder.Geocode(ctx, query)
	if err != nil {
		return Result{}, false, err
	}
	if r.cache != nil {
		r.cache.Set(query, CacheEntry{
			Lat:       result.Lat,
			Lng:       result.Lng,
			Found:     result.Found,
			UpdatedAt: r.now(),
		})
	}
	return result, false, nil
}
