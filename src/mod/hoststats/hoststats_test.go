package hoststats

import (
	"testing"
	"time"
)

func TestHostStatistics(t *testing.T) {
	// Create a test statistics object
	stats := &HostStatistics{
		Hostname:     "test.example.com",
		MinBandwidth: -1,
		LastUpdated:  time.Now(),
	}

	// Test recording requests
	stats.mu.Lock()
	stats.TotalRequests = 100
	stats.CachedRequests = 75
	stats.CacheMisses = 25
	stats.mu.Unlock()

	// Calculate hit rate
	stats.mu.Lock()
	stats.CacheHitRate = float64(stats.CachedRequests) / float64(stats.TotalRequests) * 100.0
	stats.mu.Unlock()

	if stats.CacheHitRate != 75.0 {
		t.Errorf("Expected cache hit rate 75.0, got %f", stats.CacheHitRate)
	}

	// Test traffic recording
	stats.mu.Lock()
	stats.BytesSent = 1024 * 1024 // 1MB
	stats.BytesReceived = 512 * 1024 // 512KB
	stats.mu.Unlock()

	if stats.BytesSent != 1024*1024 {
		t.Errorf("Expected bytes sent 1048576, got %d", stats.BytesSent)
	}

	// Test bandwidth sample
	sample := BandwidthSample{
		Timestamp:      time.Now(),
		BytesPerSecond: 1000000, // 1MB/s
	}

	stats.mu.Lock()
	stats.BandwidthSamples = append(stats.BandwidthSamples, sample)
	stats.CurrentBandwidth = sample.BytesPerSecond
	stats.MaxBandwidth = sample.BytesPerSecond
	stats.mu.Unlock()

	if stats.CurrentBandwidth != 1000000 {
		t.Errorf("Expected current bandwidth 1000000, got %d", stats.CurrentBandwidth)
	}
}

func TestCollectorRecordRequest(t *testing.T) {
	collector := &Collector{
		stats: make(map[string]*HostStatistics),
	}

	hostname := "test.example.com"

	// Record a cached request
	collector.RecordRequest(hostname, true)

	// Check that stats were created
	stats := collector.GetHostStats(hostname)
	if stats == nil {
		t.Fatal("Expected statistics to be created")
	}

	if stats.TotalRequests != 1 {
		t.Errorf("Expected total requests 1, got %d", stats.TotalRequests)
	}

	if stats.CachedRequests != 1 {
		t.Errorf("Expected cached requests 1, got %d", stats.CachedRequests)
	}

	// Record a cache miss
	collector.RecordRequest(hostname, false)

	stats = collector.GetHostStats(hostname)
	if stats.TotalRequests != 2 {
		t.Errorf("Expected total requests 2, got %d", stats.TotalRequests)
	}

	if stats.CacheMisses != 1 {
		t.Errorf("Expected cache misses 1, got %d", stats.CacheMisses)
	}
}

func TestCollectorRecordTraffic(t *testing.T) {
	collector := &Collector{
		stats: make(map[string]*HostStatistics),
	}

	hostname := "test.example.com"

	// Record traffic
	collector.RecordTraffic(hostname, 1024, 512)

	stats := collector.GetHostStats(hostname)
	if stats == nil {
		t.Fatal("Expected statistics to be created")
	}

	if stats.BytesSent != 1024 {
		t.Errorf("Expected bytes sent 1024, got %d", stats.BytesSent)
	}

	if stats.BytesReceived != 512 {
		t.Errorf("Expected bytes received 512, got %d", stats.BytesReceived)
	}
}

func TestCollectorRecordCacheData(t *testing.T) {
	collector := &Collector{
		stats: make(map[string]*HostStatistics),
	}

	hostname := "test.example.com"

	// Record cache data
	collector.RecordCacheData(hostname, 2048, 5)

	stats := collector.GetHostStats(hostname)
	if stats == nil {
		t.Fatal("Expected statistics to be created")
	}

	if stats.CachedDataSize != 2048 {
		t.Errorf("Expected cached data size 2048, got %d", stats.CachedDataSize)
	}

	if stats.CachedObjects != 5 {
		t.Errorf("Expected cached objects 5, got %d", stats.CachedObjects)
	}
}

func TestCollectorResetHostStats(t *testing.T) {
	collector := &Collector{
		stats: make(map[string]*HostStatistics),
	}

	hostname := "test.example.com"

	// Record some data
	collector.RecordRequest(hostname, true)
	collector.RecordTraffic(hostname, 1024, 512)
	collector.RecordCacheData(hostname, 2048, 5)

	// Reset statistics
	collector.ResetHostStats(hostname)

	stats := collector.GetHostStats(hostname)
	if stats == nil {
		t.Fatal("Expected statistics to exist after reset")
	}

	if stats.TotalRequests != 0 {
		t.Errorf("Expected total requests to be reset to 0, got %d", stats.TotalRequests)
	}

	if stats.BytesSent != 0 {
		t.Errorf("Expected bytes sent to be reset to 0, got %d", stats.BytesSent)
	}

	if stats.CachedDataSize != 0 {
		t.Errorf("Expected cached data size to be reset to 0, got %d", stats.CachedDataSize)
	}
}
