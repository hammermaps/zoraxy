package hoststats

import (
	"encoding/json"
	"sync"
	"time"

	"imuslab.com/zoraxy/mod/database"
)

/*
	Host Statistics Package

	This package tracks per-host statistics including:
	- Request counts
	- Cached data size
	- Traffic (bytes sent/received)
	- Bandwidth (current, max, min)
*/

const (
	BANDWIDTH_SAMPLE_INTERVAL = 5 * time.Second // Sample bandwidth every 5 seconds
	MAX_BANDWIDTH_SAMPLES     = 17280           // Keep 24 hours of samples (24h * 60min * 60sec / 5sec = 17280)
)

// HostStatistics holds statistics for a single host
type HostStatistics struct {
	Hostname string `json:"hostname"`

	// Request counters
	TotalRequests  int64 `json:"total_requests"`
	CachedRequests int64 `json:"cached_requests"`
	CacheMisses    int64 `json:"cache_misses"`
	CacheHitRate   float64 `json:"cache_hit_rate"` // Percentage

	// Cache statistics
	CachedDataSize int64 `json:"cached_data_size"` // Total size of cached data in bytes
	CachedObjects  int64 `json:"cached_objects"`   // Number of cached objects

	// Traffic statistics
	BytesSent     int64 `json:"bytes_sent"`     // Total bytes sent to clients
	BytesReceived int64 `json:"bytes_received"` // Total bytes received from upstream

	// Bandwidth statistics (bytes per second)
	CurrentBandwidth    int64 `json:"current_bandwidth"`     // Current bandwidth usage
	MaxBandwidth        int64 `json:"max_bandwidth"`         // Maximum bandwidth observed
	MinBandwidth        int64 `json:"min_bandwidth"`         // Minimum bandwidth observed (non-zero)
	MinBandwidthRecorded bool  `json:"min_bandwidth_recorded"` // Whether MinBandwidth has been set

	// Time-series bandwidth data for graphical display
	BandwidthSamples []BandwidthSample `json:"bandwidth_samples"`

	// Last update timestamp
	LastUpdated time.Time `json:"last_updated"`

	mu sync.RWMutex `json:"-"`
}

// BandwidthSample represents a bandwidth measurement at a specific time
type BandwidthSample struct {
	Timestamp time.Time `json:"timestamp"`
	BytesPerSecond int64 `json:"bytes_per_second"`
}

// Collector manages statistics for all hosts
type Collector struct {
	stats    map[string]*HostStatistics // Map of hostname to statistics
	mu       sync.RWMutex
	database *database.Database
	stopChan chan bool
	ticker   *time.Ticker
}

// CollectorOption holds configuration for the collector
type CollectorOption struct {
	Database *database.Database
}

// NewCollector creates a new host statistics collector
func NewCollector(option CollectorOption) (*Collector, error) {
	option.Database.NewTable("hoststats")

	collector := &Collector{
		stats:    make(map[string]*HostStatistics),
		database: option.Database,
		stopChan: make(chan bool),
	}

	// Load existing statistics from database
	collector.loadFromDatabase()

	// Start bandwidth sampling
	collector.startBandwidthSampling()

	// Schedule daily persistence
	collector.scheduleDailyPersistence()

	return collector, nil
}

// GetHostStats returns statistics for a specific host
func (c *Collector) GetHostStats(hostname string) *HostStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats, exists := c.stats[hostname]
	if !exists {
		return nil
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	// Return a copy to avoid data races
	statsCopy := *stats
	statsCopy.BandwidthSamples = make([]BandwidthSample, len(stats.BandwidthSamples))
	copy(statsCopy.BandwidthSamples, stats.BandwidthSamples)

	return &statsCopy
}

// GetAllHostStats returns statistics for all hosts
func (c *Collector) GetAllHostStats() map[string]*HostStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*HostStatistics)
	for hostname, stats := range c.stats {
		stats.mu.RLock()
		statsCopy := *stats
		statsCopy.BandwidthSamples = make([]BandwidthSample, len(stats.BandwidthSamples))
		copy(statsCopy.BandwidthSamples, stats.BandwidthSamples)
		result[hostname] = &statsCopy
		stats.mu.RUnlock()
	}

	return result
}

// RecordRequest records a request for a host
func (c *Collector) RecordRequest(hostname string, cached bool) {
	c.mu.Lock()
	stats, exists := c.stats[hostname]
	if !exists {
		stats = &HostStatistics{
			Hostname:             hostname,
			LastUpdated:          time.Now(),
			MinBandwidthRecorded: false,
		}
		c.stats[hostname] = stats
	}
	c.mu.Unlock()

	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.TotalRequests++
	if cached {
		stats.CachedRequests++
	} else {
		stats.CacheMisses++
	}

	// Calculate cache hit rate
	if stats.TotalRequests > 0 {
		stats.CacheHitRate = float64(stats.CachedRequests) / float64(stats.TotalRequests) * 100.0
	}

	stats.LastUpdated = time.Now()
}

// RecordTraffic records traffic for a host
func (c *Collector) RecordTraffic(hostname string, bytesSent, bytesReceived int64) {
	c.mu.Lock()
	stats, exists := c.stats[hostname]
	if !exists {
		stats = &HostStatistics{
			Hostname:             hostname,
			LastUpdated:          time.Now(),
			MinBandwidthRecorded: false,
		}
		c.stats[hostname] = stats
	}
	c.mu.Unlock()

	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.BytesSent += bytesSent
	stats.BytesReceived += bytesReceived
	stats.LastUpdated = time.Now()
}

// RecordCacheData records cache data statistics
func (c *Collector) RecordCacheData(hostname string, dataSizeDelta int64, objectsDelta int64) {
	c.mu.Lock()
	stats, exists := c.stats[hostname]
	if !exists {
		stats = &HostStatistics{
			Hostname:             hostname,
			LastUpdated:          time.Now(),
			MinBandwidthRecorded: false,
		}
		c.stats[hostname] = stats
	}
	c.mu.Unlock()

	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.CachedDataSize += dataSizeDelta
	stats.CachedObjects += objectsDelta
	stats.LastUpdated = time.Now()
}

// startBandwidthSampling starts periodic bandwidth sampling
func (c *Collector) startBandwidthSampling() {
	c.ticker = time.NewTicker(BANDWIDTH_SAMPLE_INTERVAL)

	go func() {
		lastSampleTime := time.Now()
		lastBytesSent := make(map[string]int64)
		lastBytesReceived := make(map[string]int64)

		for {
			select {
			case <-c.ticker.C:
				now := time.Now()
				elapsed := now.Sub(lastSampleTime).Seconds()

				c.mu.RLock()
				for hostname, stats := range c.stats {
					stats.mu.Lock()

					// Calculate bandwidth
					bytesSent := stats.BytesSent
					bytesReceived := stats.BytesReceived

					lastSent := lastBytesSent[hostname]
					lastReceived := lastBytesReceived[hostname]

					deltaBytes := (bytesSent - lastSent) + (bytesReceived - lastReceived)
					bandwidth := int64(float64(deltaBytes) / elapsed)

					// Update current bandwidth
					stats.CurrentBandwidth = bandwidth

					// Update max bandwidth
					if bandwidth > stats.MaxBandwidth {
						stats.MaxBandwidth = bandwidth
					}

					// Update min bandwidth (ignore zero values)
					if bandwidth > 0 && (!stats.MinBandwidthRecorded || bandwidth < stats.MinBandwidth) {
						stats.MinBandwidth = bandwidth
						stats.MinBandwidthRecorded = true
					}

					// Add bandwidth sample
					sample := BandwidthSample{
						Timestamp:      now,
						BytesPerSecond: bandwidth,
					}
					stats.BandwidthSamples = append(stats.BandwidthSamples, sample)

					// Keep only recent samples
					if len(stats.BandwidthSamples) > MAX_BANDWIDTH_SAMPLES {
						stats.BandwidthSamples = stats.BandwidthSamples[len(stats.BandwidthSamples)-MAX_BANDWIDTH_SAMPLES:]
					}

					lastBytesSent[hostname] = bytesSent
					lastBytesReceived[hostname] = bytesReceived

					stats.mu.Unlock()
				}
				c.mu.RUnlock()

				lastSampleTime = now

			case <-c.stopChan:
				c.ticker.Stop()
				return
			}
		}
	}()
}

// scheduleDailyPersistence saves statistics to database daily at midnight
func (c *Collector) scheduleDailyPersistence() {
	go func() {
		for {
			// Calculate duration until next midnight
			now := time.Now()
			midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			duration := midnight.Sub(now)

			select {
			case <-time.After(duration):
				c.saveToDatabase()
			case <-c.stopChan:
				return
			}
		}
	}()
}

// saveToDatabase saves all statistics to the database
func (c *Collector) saveToDatabase() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for hostname, stats := range c.stats {
		stats.mu.RLock()
		data, err := json.Marshal(stats)
		stats.mu.RUnlock()

		if err != nil {
			continue
		}

		c.database.Write("hoststats", hostname, string(data))
	}
}

// loadFromDatabase loads all statistics from the database
func (c *Collector) loadFromDatabase() {
	// List all entries in hoststats table
	entries, err := c.database.ListTable("hoststats")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if len(entry) < 2 {
			continue
		}
		
		key := string(entry[0])
		
		var statsJSON string
		err := c.database.Read("hoststats", key, &statsJSON)
		if err != nil {
			continue
		}

		var stats HostStatistics
		err = json.Unmarshal([]byte(statsJSON), &stats)
		if err != nil {
			continue
		}

		c.stats[stats.Hostname] = &stats
	}
}

// ResetHostStats resets statistics for a specific host
func (c *Collector) ResetHostStats(hostname string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats, exists := c.stats[hostname]
	if !exists {
		return
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.TotalRequests = 0
	stats.CachedRequests = 0
	stats.CacheMisses = 0
	stats.CacheHitRate = 0
	stats.CachedDataSize = 0
	stats.CachedObjects = 0
	stats.BytesSent = 0
	stats.BytesReceived = 0
	stats.CurrentBandwidth = 0
	stats.MaxBandwidth = 0
	stats.MinBandwidth = 0
	stats.MinBandwidthRecorded = false
	stats.BandwidthSamples = []BandwidthSample{}
	stats.LastUpdated = time.Now()
}

// Close stops the collector and saves all data
func (c *Collector) Close() {
	close(c.stopChan)
	c.saveToDatabase()
}
