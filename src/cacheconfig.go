package main

import (
	"encoding/json"
	"os"
	"regexp"
	"time"

	"imuslab.com/zoraxy/mod/cache"
	"imuslab.com/zoraxy/mod/cachemiddleware"
	"imuslab.com/zoraxy/mod/cacheworker"
	"imuslab.com/zoraxy/mod/optimizer"
)

const (
	CONF_CACHE_CONFIG = CONF_FOLDER + "/cache_conf.json"
	CONF_CACHE_STORE  = CONF_FOLDER + "/cache"
)

// CacheConfiguration holds the configuration for the cache system
type CacheConfiguration struct {
	Enabled bool   `json:"enabled"`
	Backend string `json:"backend"` // "fs", "redis", "varnish"

	// Filesystem backend settings
	FS struct {
		Root       string `json:"root"`
		ShardDepth int    `json:"shard_depth"`
	} `json:"fs"`

	// Redis backend settings
	Redis struct {
		Addr     string `json:"addr"`
		Password string `json:"password"`
		DB       int    `json:"db"`
	} `json:"redis"`

	// Varnish backend settings
	Varnish struct {
		Endpoints []string `json:"endpoints"`
	} `json:"varnish"`

	// Cache settings
	TTL          int   `json:"ttl"`           // Default TTL in seconds
	MaxCacheSize int64 `json:"max_cache_size"` // Maximum cache size in bytes

	// Optimization settings
	Optimize struct {
		Mode       string `json:"mode"` // "sync", "async", "disabled"
		MinifyCSS  bool   `json:"minify_css"`
		MinifyJS   bool   `json:"minify_js"`
		MinifyHTML bool   `json:"minify_html"`
		CompressBr bool   `json:"compress_brotli"`
		CompressGz bool   `json:"compress_gzip"`
	} `json:"optimize"`

	// Cacheable paths (regex patterns)
	CacheablePaths []string `json:"cacheable_paths"`

	// Admin secret for cache management endpoints
	AdminSecret string `json:"admin_secret"`
}

// DefaultCacheConfiguration returns the default cache configuration
func DefaultCacheConfiguration() *CacheConfiguration {
	config := &CacheConfiguration{
		Enabled:      false,
		Backend:      "fs",
		TTL:          3600,
		MaxCacheSize: 104857600, // 100MB
	}

	config.FS.Root = CONF_CACHE_STORE
	config.FS.ShardDepth = 2

	config.Optimize.Mode = "disabled"
	config.Optimize.MinifyCSS = true
	config.Optimize.MinifyJS = true
	config.Optimize.MinifyHTML = true
	config.Optimize.CompressBr = true
	config.Optimize.CompressGz = false // Prefer brotli over gzip

	config.CacheablePaths = []string{
		`^/static/.*\.(js|css|jpg|jpeg|png|gif|svg|ico|woff|woff2|ttf|eot)$`,
	}

	return config
}

// LoadCacheConfiguration loads cache configuration from file
func LoadCacheConfiguration() (*CacheConfiguration, error) {
	if _, err := os.Stat(CONF_CACHE_CONFIG); os.IsNotExist(err) {
		// Config file doesn't exist, create default
		config := DefaultCacheConfiguration()
		if err := SaveCacheConfiguration(config); err != nil {
			return nil, err
		}
		return config, nil
	}

	data, err := os.ReadFile(CONF_CACHE_CONFIG)
	if err != nil {
		return nil, err
	}

	config := DefaultCacheConfiguration()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveCacheConfiguration saves cache configuration to file
func SaveCacheConfiguration(config *CacheConfiguration) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(CONF_CACHE_CONFIG, data, 0644)
}

// BuildCacheStore creates a cache store from configuration
func BuildCacheStore(config *CacheConfiguration) (cache.CacheStore, error) {
	switch config.Backend {
	case "fs":
		return cache.NewFSStore(config.FS.Root, config.FS.ShardDepth)

	case "redis":
		return cache.NewRedisStore(cache.RedisStoreConfig{
			Addr:     config.Redis.Addr,
			Password: config.Redis.Password,
			DB:       config.Redis.DB,
			Prefix:   "zoraxy:cache:",
			MaxSize:  config.MaxCacheSize,
		})

	case "varnish":
		return cache.NewVarnishStore(cache.VarnishStoreConfig{
			Endpoints: config.Varnish.Endpoints,
		})

	default:
		// Default to filesystem
		return cache.NewFSStore(config.FS.Root, config.FS.ShardDepth)
	}
}

// BuildOptimizationPipeline creates an optimization pipeline from configuration
func BuildOptimizationPipeline(config *CacheConfiguration) *optimizer.Pipeline {
	if config.Optimize.Mode == "disabled" {
		return nil
	}

	pipeline := optimizer.NewPipeline()

	// Add minification transforms
	if config.Optimize.MinifyCSS || config.Optimize.MinifyJS || config.Optimize.MinifyHTML {
		minifyConfig := optimizer.MinifyConfig{
			HTML: config.Optimize.MinifyHTML,
			CSS:  config.Optimize.MinifyCSS,
			JS:   config.Optimize.MinifyJS,
			JSON: true,
			SVG:  true,
			XML:  false,
		}
		pipeline.AddTransform(optimizer.MinifyTransform(minifyConfig))
	}

	// Add compression transforms
	if config.Optimize.CompressBr {
		pipeline.AddTransform(optimizer.BrotliTransform(6))
	} else if config.Optimize.CompressGz {
		pipeline.AddTransform(optimizer.GzipTransform(-1)) // Default compression
	}

	return pipeline
}

// BuildCacheMiddlewareConfig creates middleware configuration from cache configuration
func BuildCacheMiddlewareConfig(config *CacheConfiguration, store cache.CacheStore, worker *cacheworker.Worker) (cachemiddleware.Config, error) {
	// Compile cacheable path patterns
	var patterns []*regexp.Regexp
	for _, pattern := range config.CacheablePaths {
		re, err := regexp.Compile(pattern)
		if err != nil {
			SystemWideLogger.Println("Invalid cache path pattern", pattern, ":", err)
			continue
		}
		patterns = append(patterns, re)
	}

	// Build optimization pipeline
	pipeline := BuildOptimizationPipeline(config)

	// Determine optimization mode
	var optMode cachemiddleware.OptimizationMode
	switch config.Optimize.Mode {
	case "sync":
		optMode = cachemiddleware.OptimizationSync
	case "async":
		optMode = cachemiddleware.OptimizationAsync
	default:
		optMode = cachemiddleware.OptimizationDisabled
	}

	middlewareConfig := cachemiddleware.Config{
		Enabled:              config.Enabled,
		Store:                store,
		KeyGenerator:         cache.NewKeyGenerator(),
		CacheablePaths:       patterns,
		DefaultTTL:           time.Duration(config.TTL) * time.Second,
		MaxCacheSize:         config.MaxCacheSize,
		OptimizationMode:     optMode,
		OptimizationPipeline: pipeline,
		WorkerQueue:          worker,
		OnCacheEvent:         handleCacheEvent,
	}

	return middlewareConfig, nil
}

// handleCacheEvent is called when cache events occur
func handleCacheEvent(hostname string, eventType string, size int64) {
	if hostStatsCollector == nil {
		return
	}

	switch eventType {
	case "hit":
		hostStatsCollector.RecordRequest(hostname, true)
	case "miss":
		hostStatsCollector.RecordRequest(hostname, false)
	case "put":
		hostStatsCollector.RecordCacheData(hostname, size, 1)
	case "traffic":
		// Record bytes sent (traffic out)
		hostStatsCollector.RecordTraffic(hostname, size, 0)
	}
}
