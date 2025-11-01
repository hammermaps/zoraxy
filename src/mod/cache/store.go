package cache

import (
	"context"
	"io"
	"time"
)

// CacheStore defines the interface for cache storage backends
type CacheStore interface {
	// Get retrieves a cached response by key
	// Returns the body reader, metadata, found flag, and any error
	Get(ctx context.Context, key string) (io.ReadCloser, *Meta, bool, error)

	// Put stores a response in the cache
	Put(ctx context.Context, key string, body io.Reader, meta *Meta) error

	// Delete removes a cached response by key
	Delete(ctx context.Context, key string) error

	// PurgePrefix removes all cached responses with keys matching the prefix
	PurgePrefix(ctx context.Context, prefix string) error

	// Close cleanly shuts down the cache store
	Close() error
}

// Meta contains metadata about a cached response
type Meta struct {
	// ContentType is the MIME type of the response
	ContentType string

	// Encoding specifies the content encoding (e.g., "gzip", "br")
	Encoding string

	// Size is the size of the cached content in bytes
	Size int64

	// ETag is the entity tag for cache validation
	ETag string

	// TTL is the time-to-live for this cache entry
	TTL time.Duration

	// CachedAt is when this entry was cached
	CachedAt time.Time

	// StatusCode is the HTTP status code of the cached response
	StatusCode int

	// Headers stores additional HTTP headers to preserve
	Headers map[string]string
}

// IsExpired checks if the cache entry has expired
func (m *Meta) IsExpired() bool {
	if m.TTL <= 0 {
		return false // No expiration
	}
	return time.Since(m.CachedAt) > m.TTL
}

// Age returns the age of the cache entry in seconds
func (m *Meta) Age() int64 {
	return int64(time.Since(m.CachedAt).Seconds())
}
