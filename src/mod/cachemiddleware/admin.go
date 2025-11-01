package cachemiddleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"imuslab.com/zoraxy/mod/cache"
	"imuslab.com/zoraxy/mod/utils"
)

// AdminHandler provides HTTP endpoints for cache administration
type AdminHandler struct {
	middleware  *Middleware
	store       cache.CacheStore
	adminSecret string
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(middleware *Middleware, store cache.CacheStore, adminSecret string) *AdminHandler {
	return &AdminHandler{
		middleware:  middleware,
		store:       store,
		adminSecret: adminSecret,
	}
}

// authenticate checks if the request is authorized
func (ah *AdminHandler) authenticate(r *http.Request) bool {
	if ah.adminSecret == "" {
		// No auth required
		return true
	}

	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return token == ah.adminSecret
	}

	// Check query parameter
	secret := r.URL.Query().Get("secret")
	return secret == ah.adminSecret
}

// HandlePurge handles cache purge requests
func (ah *AdminHandler) HandlePurge(w http.ResponseWriter, r *http.Request) {
	if !ah.authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Key string `json:"key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendErrorResponse(w, "Invalid request body")
		return
	}

	if req.Key == "" {
		utils.SendErrorResponse(w, "Key is required")
		return
	}

	// Delete from cache
	err := ah.store.Delete(r.Context(), req.Key)
	if err != nil {
		utils.SendErrorResponse(w, "Failed to purge cache: "+err.Error())
		return
	}

	utils.SendJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Cache entry purged successfully",
		"key":     req.Key,
	})
}

// HandlePurgePrefix handles cache prefix purge requests
func (ah *AdminHandler) HandlePurgePrefix(w http.ResponseWriter, r *http.Request) {
	if !ah.authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Prefix string `json:"prefix"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendErrorResponse(w, "Invalid request body")
		return
	}

	if req.Prefix == "" {
		utils.SendErrorResponse(w, "Prefix is required")
		return
	}

	// Purge prefix from cache
	err := ah.store.PurgePrefix(r.Context(), req.Prefix)
	if err != nil {
		utils.SendErrorResponse(w, "Failed to purge cache prefix: "+err.Error())
		return
	}

	utils.SendJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Cache entries purged successfully",
		"prefix":  req.Prefix,
	})
}

// HandleStatus handles cache status requests
func (ah *AdminHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if !ah.authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := ah.middleware.GetStats()

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(stats.Hits) / float64(total) * 100
	}

	response := map[string]interface{}{
		"enabled": ah.middleware.config.Enabled,
		"backend": getBackendType(ah.store),
		"stats": map[string]interface{}{
			"hits":     stats.Hits,
			"misses":   stats.Misses,
			"puts":     stats.Puts,
			"errors":   stats.Errors,
			"bypasses": stats.Bypasses,
			"hit_rate": hitRate,
		},
		"config": map[string]interface{}{
			"optimization_mode": ah.middleware.config.OptimizationMode,
			"default_ttl":       ah.middleware.config.DefaultTTL.String(),
			"max_cache_size":    ah.middleware.config.MaxCacheSize,
		},
	}

	utils.SendJSONResponse(w, response)
}

// HandleBan handles Varnish BAN requests
func (ah *AdminHandler) HandleBan(w http.ResponseWriter, r *http.Request) {
	if !ah.authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if backend is Varnish
	varnishStore, ok := ah.store.(*cache.VarnishStore)
	if !ok {
		utils.SendErrorResponse(w, "BAN is only supported for Varnish backend")
		return
	}

	var req struct {
		Expression string `json:"expression"`
		Prefix     string `json:"prefix"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendErrorResponse(w, "Invalid request body")
		return
	}

	var err error
	if req.Expression != "" {
		// Custom BAN expression
		err = varnishStore.Ban(r.Context(), req.Expression)
	} else if req.Prefix != "" {
		// Prefix-based purge
		err = ah.store.PurgePrefix(r.Context(), req.Prefix)
	} else {
		utils.SendErrorResponse(w, "Either expression or prefix is required")
		return
	}

	if err != nil {
		utils.SendErrorResponse(w, "Failed to execute BAN: "+err.Error())
		return
	}

	utils.SendJSONResponse(w, map[string]interface{}{
		"success":    true,
		"message":    "BAN executed successfully",
		"expression": req.Expression,
		"prefix":     req.Prefix,
	})
}

// getBackendType returns a string representation of the cache backend type
func getBackendType(store cache.CacheStore) string {
	switch store.(type) {
	case *cache.FSStore:
		return "filesystem"
	case *cache.RedisStore:
		return "redis"
	case *cache.VarnishStore:
		return "varnish"
	default:
		return "unknown"
	}
}
