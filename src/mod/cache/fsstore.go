package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FSStore implements CacheStore using the filesystem
type FSStore struct {
	rootDir    string
	shardDepth int
	mu         sync.RWMutex
}

// NewFSStore creates a new filesystem-based cache store
func NewFSStore(rootDir string, shardDepth int) (*FSStore, error) {
	if shardDepth < 0 || shardDepth > 4 {
		shardDepth = 2 // Default to 2-level sharding
	}

	// Create root directory if it doesn't exist
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &FSStore{
		rootDir:    rootDir,
		shardDepth: shardDepth,
	}, nil
}

// Get retrieves a cached response from the filesystem
func (fs *FSStore) Get(ctx context.Context, key string) (io.ReadCloser, *Meta, bool, error) {
	dataPath := fs.getDataPath(key)
	metaPath := fs.getMetaPath(key)

	// Check if files exist
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return nil, nil, false, nil
	}

	// Read metadata
	meta, err := fs.readMeta(metaPath)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Check expiration
	if meta.IsExpired() {
		// Clean up expired entry
		fs.Delete(ctx, key)
		return nil, nil, false, nil
	}

	// Open data file
	file, err := os.Open(dataPath)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to open cache file: %w", err)
	}

	return file, meta, true, nil
}

// Put stores a response in the filesystem cache
func (fs *FSStore) Put(ctx context.Context, key string, body io.Reader, meta *Meta) error {
	dataPath := fs.getDataPath(key)
	metaPath := fs.getMetaPath(key)

	// Create directory structure
	dir := filepath.Dir(dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write data to temporary file first (atomic write)
	tmpDataPath := dataPath + ".tmp"
	tmpFile, err := os.Create(tmpDataPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpDataPath) // Clean up temp file on error

	// Copy data to temp file
	written, err := io.Copy(tmpFile, body)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write cache data: %w", err)
	}
	tmpFile.Close()

	// Update metadata with actual size
	meta.Size = written

	// Write metadata
	if err := fs.writeMeta(metaPath, meta); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpDataPath, dataPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Delete removes a cached entry from the filesystem
func (fs *FSStore) Delete(ctx context.Context, key string) error {
	dataPath := fs.getDataPath(key)
	metaPath := fs.getMetaPath(key)

	// Remove both files, ignore errors if files don't exist
	os.Remove(dataPath)
	os.Remove(metaPath)

	return nil
}

// PurgePrefix removes all cache entries with keys starting with the prefix
func (fs *FSStore) PurgePrefix(ctx context.Context, prefix string) error {
	// Walk the cache directory and delete matching entries
	// This is a simple implementation; for production, consider maintaining an index
	return filepath.Walk(fs.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process data files (not metadata)
		if !strings.HasSuffix(path, ".data") {
			return nil
		}

		// Extract key from path and check prefix
		// This is simplified; in production, you'd need a proper key->path mapping
		if strings.Contains(path, prefix) {
			key := filepath.Base(strings.TrimSuffix(path, ".data"))
			fs.Delete(ctx, key)
		}

		return nil
	})
}

// Close cleanly shuts down the filesystem store
func (fs *FSStore) Close() error {
	// No resources to clean up for filesystem store
	return nil
}

// getDataPath returns the filesystem path for cached data
func (fs *FSStore) getDataPath(key string) string {
	return fs.getShardedPath(key, ".data")
}

// getMetaPath returns the filesystem path for metadata
func (fs *FSStore) getMetaPath(key string) string {
	return fs.getShardedPath(key, ".meta")
}

// getShardedPath creates a sharded directory path from a key
func (fs *FSStore) getShardedPath(key string, suffix string) string {
	if fs.shardDepth == 0 {
		return filepath.Join(fs.rootDir, key+suffix)
	}

	// Create shard directories based on key prefix
	var shardParts []string
	for i := 0; i < fs.shardDepth && i*2 < len(key); i++ {
		shardParts = append(shardParts, key[i*2:i*2+2])
	}

	path := filepath.Join(fs.rootDir, filepath.Join(shardParts...))
	return filepath.Join(path, key+suffix)
}

// readMeta reads metadata from a file
func (fs *FSStore) readMeta(path string) (*Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// writeMeta writes metadata to a file
func (fs *FSStore) writeMeta(path string, meta *Meta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}
