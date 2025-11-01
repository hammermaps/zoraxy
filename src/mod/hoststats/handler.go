package hoststats

import (
	"encoding/json"
	"net/http"
	"strings"
)

// HandleGetAllHostStats returns statistics for all hosts
func (c *Collector) HandleGetAllHostStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allStats := c.GetAllHostStats()

	// Convert to JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allStats)
}

// HandleGetHostStats returns statistics for a specific host
func (c *Collector) HandleGetHostStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get hostname from query parameter
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		http.Error(w, "hostname parameter is required", http.StatusBadRequest)
		return
	}

	stats := c.GetHostStats(hostname)
	if stats == nil {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	// Convert to JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleGetHostBandwidth returns bandwidth data for a specific host
func (c *Collector) HandleGetHostBandwidth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get hostname from query parameter
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		http.Error(w, "hostname parameter is required", http.StatusBadRequest)
		return
	}

	stats := c.GetHostStats(hostname)
	if stats == nil {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	// Return bandwidth information
	bandwidthData := map[string]interface{}{
		"hostname":          stats.Hostname,
		"current_bandwidth": stats.CurrentBandwidth,
		"max_bandwidth":     stats.MaxBandwidth,
		"min_bandwidth":     stats.MinBandwidth,
		"samples":           stats.BandwidthSamples,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bandwidthData)
}

// HandleResetHostStats resets statistics for a specific host
func (c *Collector) HandleResetHostStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get hostname from query parameter
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		http.Error(w, "hostname parameter is required", http.StatusBadRequest)
		return
	}

	c.ResetHostStats(hostname)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Host statistics reset successfully",
	})
}

// HandleGetHostList returns a list of all tracked hosts with basic stats
func (c *Collector) HandleGetHostList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allStats := c.GetAllHostStats()

	// Create a simplified list
	type HostSummary struct {
		Hostname       string  `json:"hostname"`
		TotalRequests  int64   `json:"total_requests"`
		CacheHitRate   float64 `json:"cache_hit_rate"`
		BytesSent      int64   `json:"bytes_sent"`
		BytesReceived  int64   `json:"bytes_received"`
		MaxBandwidth   int64   `json:"max_bandwidth"`
	}

	summaries := make([]HostSummary, 0, len(allStats))
	for _, stats := range allStats {
		summaries = append(summaries, HostSummary{
			Hostname:      stats.Hostname,
			TotalRequests: stats.TotalRequests,
			CacheHitRate:  stats.CacheHitRate,
			BytesSent:     stats.BytesSent,
			BytesReceived: stats.BytesReceived,
			MaxBandwidth:  stats.MaxBandwidth,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// GetHostnameFromRequest extracts hostname from request
func GetHostnameFromRequest(r *http.Request) string {
	hostname := r.Host
	// Remove port if present
	if idx := strings.Index(hostname, ":"); idx != -1 {
		hostname = hostname[:idx]
	}
	return hostname
}
