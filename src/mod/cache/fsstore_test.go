package cache

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFSStore_PutAndGet(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store
	store, err := NewFSStore(tmpDir, 2)
	if err != nil {
		t.Fatalf("Failed to create FSStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key-123"
	testData := []byte("Hello, World!")
	
	meta := &Meta{
		ContentType: "text/plain",
		Size:        int64(len(testData)),
		StatusCode:  200,
		TTL:         1 * time.Hour,
		CachedAt:    time.Now(),
	}

	// Put data
	err = store.Put(ctx, key, bytes.NewReader(testData), meta)
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	// Get data
	reader, gotMeta, found, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}
	if !found {
		t.Fatal("Expected data to be found")
	}
	defer reader.Close()

	// Verify metadata
	if gotMeta.ContentType != meta.ContentType {
		t.Errorf("Expected ContentType %s, got %s", meta.ContentType, gotMeta.ContentType)
	}
	if gotMeta.StatusCode != meta.StatusCode {
		t.Errorf("Expected StatusCode %d, got %d", meta.StatusCode, gotMeta.StatusCode)
	}

	// Verify data
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	if !bytes.Equal(buf.Bytes(), testData) {
		t.Errorf("Expected data %s, got %s", testData, buf.Bytes())
	}
}

func TestFSStore_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFSStore(tmpDir, 2)
	if err != nil {
		t.Fatalf("Failed to create FSStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key-456"
	testData := []byte("Test data")
	
	meta := &Meta{
		ContentType: "text/plain",
		TTL:         1 * time.Hour,
		CachedAt:    time.Now(),
	}

	// Put and verify
	store.Put(ctx, key, bytes.NewReader(testData), meta)
	_, _, found, _ := store.Get(ctx, key)
	if !found {
		t.Fatal("Expected data to be found after put")
	}

	// Delete
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify deletion
	_, _, found, _ = store.Get(ctx, key)
	if found {
		t.Fatal("Expected data to be deleted")
	}
}

func TestFSStore_Expiration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFSStore(tmpDir, 2)
	if err != nil {
		t.Fatalf("Failed to create FSStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key-expired"
	testData := []byte("Expired data")
	
	meta := &Meta{
		ContentType: "text/plain",
		TTL:         100 * time.Millisecond, // Very short TTL
		CachedAt:    time.Now(),
	}

	// Put data
	store.Put(ctx, key, bytes.NewReader(testData), meta)

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Try to get expired data
	_, _, found, _ := store.Get(ctx, key)
	if found {
		t.Fatal("Expected expired data to not be found")
	}
}

func TestFSStore_Sharding(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFSStore(tmpDir, 2)
	if err != nil {
		t.Fatalf("Failed to create FSStore: %v", err)
	}
	defer store.Close()

	key := "abcd1234567890"
	dataPath := store.getDataPath(key)

	// Verify path contains shard directories
	expectedShards := filepath.Join(tmpDir, "ab", "cd", key+".data")
	if dataPath != expectedShards {
		t.Errorf("Expected sharded path %s, got %s", expectedShards, dataPath)
	}
}

func TestCacheMeta_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		cachedAt time.Time
		want     bool
	}{
		{
			name:     "not expired",
			ttl:      1 * time.Hour,
			cachedAt: time.Now(),
			want:     false,
		},
		{
			name:     "expired",
			ttl:      1 * time.Millisecond,
			cachedAt: time.Now().Add(-1 * time.Second),
			want:     true,
		},
		{
			name:     "no expiration",
			ttl:      0,
			cachedAt: time.Now().Add(-1 * time.Hour),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &Meta{
				TTL:      tt.ttl,
				CachedAt: tt.cachedAt,
			}
			if got := meta.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
