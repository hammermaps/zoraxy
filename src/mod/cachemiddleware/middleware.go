package cachemiddleware

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"imuslab.com/zoraxy/mod/cache"
	"imuslab.com/zoraxy/mod/optimizer"
)

// Config holds configuration for the cache middleware
type Config struct {
	// Enabled controls whether caching is active
	Enabled bool

	// Store is the cache backend
	Store cache.CacheStore

	// KeyGenerator generates cache keys from requests
	KeyGenerator *cache.KeyGenerator

	// CacheablePaths are regex patterns for paths that should be cached
	CacheablePaths []*regexp.Regexp

	// DefaultTTL is the default time-to-live for cached entries
	DefaultTTL time.Duration

	// MaxCacheSize is the maximum size in bytes for a cacheable response
	MaxCacheSize int64

	// OptimizationMode determines when optimization occurs
	OptimizationMode OptimizationMode

	// OptimizationPipeline is the pipeline of transforms to apply
	OptimizationPipeline *optimizer.Pipeline

	// WorkerQueue is the queue for async optimization jobs
	WorkerQueue JobQueue

	// OnCacheEvent is called when cache events occur (hit, miss, put)
	OnCacheEvent func(hostname string, eventType string, size int64)
}

// OptimizationMode specifies when optimization should occur
type OptimizationMode string

const (
	// OptimizationDisabled disables optimization
	OptimizationDisabled OptimizationMode = "disabled"

	// OptimizationSync applies optimization before caching and serving
	OptimizationSync OptimizationMode = "sync"

	// OptimizationAsync caches raw response and optimizes asynchronously
	OptimizationAsync OptimizationMode = "async"
)

// JobQueue is an interface for enqueueing optimization jobs
type JobQueue interface {
	Enqueue(job OptimizationJob) error
}

// OptimizationJob represents a job to optimize cached content
type OptimizationJob struct {
	Key      string
	Store    cache.CacheStore
	Pipeline *optimizer.Pipeline
}

// Middleware wraps an HTTP handler with caching functionality
type Middleware struct {
	config  Config
	handler http.Handler
	stats   *Stats
}

// Stats tracks cache statistics
type Stats struct {
	mu       sync.RWMutex
	Hits     int64
	Misses   int64
	Puts     int64
	Errors   int64
	Bypasses int64
}

// NewMiddleware creates a new cache middleware
func NewMiddleware(config Config, handler http.Handler) *Middleware {
	if config.KeyGenerator == nil {
		config.KeyGenerator = cache.NewKeyGenerator()
	}

	if config.DefaultTTL <= 0 {
		config.DefaultTTL = 1 * time.Hour
	}

	if config.MaxCacheSize <= 0 {
		config.MaxCacheSize = 10 * 1024 * 1024 // 10MB default
	}

	return &Middleware{
		config:  config,
		handler: handler,
		stats:   &Stats{},
	}
}

// ServeHTTP implements http.Handler
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !m.config.Enabled {
		m.handler.ServeHTTP(w, r)
		return
	}

	// Check if request is cacheable
	if !m.isCacheable(r) {
		m.stats.incrementBypasses()
		m.handler.ServeHTTP(w, r)
		return
	}

	// Generate cache key
	key := m.config.KeyGenerator.GenerateKey(r)

	// Try to get from cache
	ctx := r.Context()
	reader, meta, found, err := m.config.Store.Get(ctx, key)
	if err != nil {
		// Error reading from cache, bypass
		m.stats.incrementErrors()
		m.handler.ServeHTTP(w, r)
		return
	}

	if found {
		// Cache hit - serve from cache
		m.stats.incrementHits()
		
		// Notify about cache hit
		if m.config.OnCacheEvent != nil {
			hostname := r.Host
			m.config.OnCacheEvent(hostname, "hit", 0)
		}
		
		m.serveCachedResponse(w, r, reader, meta)
		return
	}

	// Cache miss - fetch from upstream and cache
	m.stats.incrementMisses()
	
	// Notify about cache miss
	if m.config.OnCacheEvent != nil {
		hostname := r.Host
		m.config.OnCacheEvent(hostname, "miss", 0)
	}
	
	m.fetchAndCache(w, r, key)
}

// isCacheable checks if a request should be cached
func (m *Middleware) isCacheable(r *http.Request) bool {
	// Check if request is cacheable
	if !cache.IsCacheable(r) {
		return false
	}

	// Check if path matches cacheable patterns
	if len(m.config.CacheablePaths) > 0 {
		matched := false
		for _, pattern := range m.config.CacheablePaths {
			if pattern.MatchString(r.URL.Path) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// serveCachedResponse serves a response from cache
func (m *Middleware) serveCachedResponse(w http.ResponseWriter, r *http.Request, reader io.ReadCloser, meta *cache.Meta) {
	defer reader.Close()

	// Set cache headers
	w.Header().Set("X-Cache", "HIT")
	w.Header().Set("Age", strconv.FormatInt(meta.Age(), 10))

	// Set content headers
	if meta.ContentType != "" {
		w.Header().Set("Content-Type", meta.ContentType)
	}
	if meta.Encoding != "" {
		w.Header().Set("Content-Encoding", meta.Encoding)
	}
	if meta.ETag != "" {
		w.Header().Set("ETag", meta.ETag)
	}

	// Set cache control
	remainingTTL := meta.TTL - time.Since(meta.CachedAt)
	if remainingTTL > 0 {
		w.Header().Set("Cache-Control", "public, max-age="+strconv.FormatInt(int64(remainingTTL.Seconds()), 10))
	}

	// Copy additional headers
	for k, v := range meta.Headers {
		w.Header().Set(k, v)
	}

	// Write status code
	w.WriteHeader(meta.StatusCode)

	// Stream response body and track bytes sent
	bytesSent, _ := io.Copy(w, reader)
	
	// Notify about traffic
	if m.config.OnCacheEvent != nil && bytesSent > 0 {
		hostname := r.Host
		m.config.OnCacheEvent(hostname, "traffic", bytesSent)
	}
}

// fetchAndCache fetches from upstream and caches the response
func (m *Middleware) fetchAndCache(w http.ResponseWriter, r *http.Request, key string) {
	// Create a response recorder to capture the upstream response
	recorder := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		headers:        make(http.Header),
		body:           &bytes.Buffer{},
	}

	// Call upstream handler
	m.handler.ServeHTTP(recorder, r)

	// Check if response is cacheable
	if !cache.IsResponseCacheable(recorder.statusCode, recorder.headers) {
		// Write captured response and return
		m.writeRecordedResponse(w, recorder, r)
		return
	}

	// Check size limit
	if int64(recorder.body.Len()) > m.config.MaxCacheSize {
		m.writeRecordedResponse(w, recorder, r)
		return
	}

	// Create metadata
	meta := &cache.Meta{
		ContentType: recorder.headers.Get("Content-Type"),
		StatusCode:  recorder.statusCode,
		TTL:         m.config.DefaultTTL,
		CachedAt:    time.Now(),
		Headers:     make(map[string]string),
	}

	// Extract ETag if present
	if etag := recorder.headers.Get("ETag"); etag != "" {
		meta.ETag = etag
	}

	// Preserve important headers
	for _, header := range []string{"Last-Modified", "Vary"} {
		if value := recorder.headers.Get(header); value != "" {
			meta.Headers[header] = value
		}
	}

	// Apply optimization if enabled
	bodyBytes := recorder.body.Bytes()

	switch m.config.OptimizationMode {
	case OptimizationSync:
		// Optimize synchronously before caching
		if m.config.OptimizationPipeline != nil {
			optimized, optimizedMeta, err := m.config.OptimizationPipeline.ApplyToBytes(r.Context(), bodyBytes, meta)
			if err == nil {
				bodyBytes = optimized
				meta = optimizedMeta
			}
		}

	case OptimizationAsync:
		// Cache raw response and schedule optimization
		if m.config.WorkerQueue != nil && m.config.OptimizationPipeline != nil {
			// Enqueue optimization job (non-blocking)
			m.config.WorkerQueue.Enqueue(OptimizationJob{
				Key:      key,
				Store:    m.config.Store,
				Pipeline: m.config.OptimizationPipeline,
			})
		}
	}

	// Store in cache
	err := m.config.Store.Put(r.Context(), key, bytes.NewReader(bodyBytes), meta)
	if err == nil {
		m.stats.incrementPuts()
		
		// Notify about cache put
		if m.config.OnCacheEvent != nil {
			hostname := r.Host
			m.config.OnCacheEvent(hostname, "put", int64(len(bodyBytes)))
		}
	}

	// Write response to client
	w.Header().Set("X-Cache", "MISS")
	m.writeRecordedResponse(w, recorder, r)
}

// writeRecordedResponse writes a recorded response to the client
func (m *Middleware) writeRecordedResponse(w http.ResponseWriter, recorder *responseRecorder, r *http.Request) {
	// Copy headers
	for k, values := range recorder.headers {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}

	// Write status code
	w.WriteHeader(recorder.statusCode)

	// Write body
	bodyBytes := recorder.body.Bytes()
	w.Write(bodyBytes)
	
	// Notify about traffic
	if m.config.OnCacheEvent != nil && len(bodyBytes) > 0 {
		hostname := r.Host
		m.config.OnCacheEvent(hostname, "traffic", int64(len(bodyBytes)))
	}
}

// responseRecorder captures an HTTP response
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	headers    http.Header
	body       *bytes.Buffer
}

func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	// Copy headers from the underlying ResponseWriter
	for k, v := range rr.ResponseWriter.Header() {
		rr.headers[k] = v
	}
}

func (rr *responseRecorder) Write(data []byte) (int, error) {
	// Write to buffer for caching
	if rr.body != nil {
		rr.body.Write(data)
	}
	// Also write to actual response
	return rr.ResponseWriter.Write(data)
}

// GetStats returns current cache statistics
func (m *Middleware) GetStats() Stats {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()
	return Stats{
		Hits:     m.stats.Hits,
		Misses:   m.stats.Misses,
		Puts:     m.stats.Puts,
		Errors:   m.stats.Errors,
		Bypasses: m.stats.Bypasses,
	}
}

func (s *Stats) incrementHits() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Hits++
}

func (s *Stats) incrementMisses() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Misses++
}

func (s *Stats) incrementPuts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Puts++
}

func (s *Stats) incrementErrors() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Errors++
}

func (s *Stats) incrementBypasses() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Bypasses++
}
