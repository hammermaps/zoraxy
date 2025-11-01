package main

import (
	"net/http"

	"imuslab.com/zoraxy/mod/cache"
	"imuslab.com/zoraxy/mod/cachemiddleware"
	"imuslab.com/zoraxy/mod/cacheworker"
	"imuslab.com/zoraxy/mod/info/logger"
)

// Global cache variables
var (
	cacheStore         cache.CacheStore
	cacheWorker        *cacheworker.Worker
	cacheMiddleware    *cachemiddleware.Middleware
	cacheAdminHandler  *cachemiddleware.AdminHandler
	cacheConfiguration *CacheConfiguration
)

// initCacheSystem initializes the cache system during startup
func initCacheSystem() error {
	SystemWideLogger.Println("Initializing cache system")

	// Load cache configuration
	config, err := LoadCacheConfiguration()
	if err != nil {
		SystemWideLogger.Println("Failed to load cache configuration:", err)
		// Use default configuration
		config = DefaultCacheConfiguration()
	}
	cacheConfiguration = config

	if !config.Enabled {
		SystemWideLogger.Println("Cache system is disabled")
		return nil
	}

	// Build cache store
	store, err := BuildCacheStore(config)
	if err != nil {
		SystemWideLogger.Println("Failed to create cache store:", err)
		return err
	}
	cacheStore = store
	SystemWideLogger.Println("Cache backend:", config.Backend)

	// Initialize worker if async optimization is enabled
	if config.Optimize.Mode == "async" {
		workerConfig := cacheworker.DefaultConfig()
		workerConfig.Logger = &loggerAdapter{SystemWideLogger}
		cacheWorker = cacheworker.NewWorker(workerConfig)
		cacheWorker.Start()
		SystemWideLogger.Println("Cache worker started with", workerConfig.WorkerCount, "workers")
	}

	SystemWideLogger.Println("Cache system initialized (TTL:", config.TTL, "s, Max size:", config.MaxCacheSize, "bytes)")
	return nil
}

// loggerAdapter adapts Zoraxy logger to cacheworker.Logger interface
type loggerAdapter struct {
	*logger.Logger
}

func (la *loggerAdapter) Printf(format string, v ...interface{}) {
	la.Println(v...)
}

// wrapHandlerWithCache wraps an HTTP handler with caching middleware
func wrapHandlerWithCache(handler http.Handler) http.Handler {
	if cacheConfiguration == nil || !cacheConfiguration.Enabled {
		return handler
	}

	// Build middleware configuration
	middlewareConfig, err := BuildCacheMiddlewareConfig(cacheConfiguration, cacheStore, cacheWorker)
	if err != nil {
		SystemWideLogger.Println("Failed to build cache middleware config:", err)
		return handler
	}

	// Create middleware
	cacheMiddleware = cachemiddleware.NewMiddleware(middlewareConfig, handler)

	// Create admin handler
	cacheAdminHandler = cachemiddleware.NewAdminHandler(cacheMiddleware, cacheStore, cacheConfiguration.AdminSecret)

	SystemWideLogger.Println("Cache middleware enabled")
	return cacheMiddleware
}

// registerCacheAPIs registers cache management API endpoints
func registerCacheAPIs(mux *http.ServeMux) {
	if cacheAdminHandler == nil {
		return
	}

	SystemWideLogger.Println("Registering cache management API endpoints")
	mux.HandleFunc("/_cache/purge", cacheAdminHandler.HandlePurge)
	mux.HandleFunc("/_cache/purge-prefix", cacheAdminHandler.HandlePurgePrefix)
	mux.HandleFunc("/_cache/status", cacheAdminHandler.HandleStatus)
	mux.HandleFunc("/_cache/ban", cacheAdminHandler.HandleBan)
}

// shutdownCacheSystem cleanly shuts down the cache system
func shutdownCacheSystem() {
	SystemWideLogger.Println("Shutting down cache system")

	if cacheWorker != nil {
		cacheWorker.Stop()
	}

	if cacheStore != nil {
		cacheStore.Close()
	}

	SystemWideLogger.Println("Cache system shut down")
}
