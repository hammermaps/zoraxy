package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements CacheStore using Redis
type RedisStore struct {
	client *redis.Client
	prefix string
	maxSize int64 // Maximum size for cached objects
}

// RedisStoreConfig holds configuration for Redis store
type RedisStoreConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string  // Key prefix for all cache entries
	MaxSize  int64   // Maximum size for cached objects (default: 10MB)
}

// NewRedisStore creates a new Redis-based cache store
func NewRedisStore(cfg RedisStoreConfig) (*RedisStore, error) {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 10 * 1024 * 1024 // 10MB default
	}

	if cfg.Prefix == "" {
		cfg.Prefix = "zoraxy:cache:"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client:  client,
		prefix:  cfg.Prefix,
		maxSize: cfg.MaxSize,
	}, nil
}

// Get retrieves a cached response from Redis
func (rs *RedisStore) Get(ctx context.Context, key string) (io.ReadCloser, *Meta, bool, error) {
	fullKey := rs.prefix + key

	// Get both data and metadata in a pipeline
	pipe := rs.client.Pipeline()
	dataCmd := pipe.Get(ctx, fullKey+":data")
	metaCmd := pipe.Get(ctx, fullKey+":meta")

	_, err := pipe.Exec(ctx)
	if err == redis.Nil {
		return nil, nil, false, nil
	}
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to get from Redis: %w", err)
	}

	// Parse metadata
	metaBytes, err := metaCmd.Bytes()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to get metadata: %w", err)
	}

	var meta Meta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, nil, false, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Check expiration
	if meta.IsExpired() {
		rs.Delete(ctx, key)
		return nil, nil, false, nil
	}

	// Get data
	dataBytes, err := dataCmd.Bytes()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to get data: %w", err)
	}

	// Return data as ReadCloser
	reader := io.NopCloser(bytes.NewReader(dataBytes))
	return reader, &meta, true, nil
}

// Put stores a response in Redis
func (rs *RedisStore) Put(ctx context.Context, key string, body io.Reader, meta *Meta) error {
	fullKey := rs.prefix + key

	// Read body into memory
	dataBytes, err := io.ReadAll(io.LimitReader(body, rs.maxSize+1))
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	// Check size limit
	if int64(len(dataBytes)) > rs.maxSize {
		return fmt.Errorf("cache entry exceeds maximum size: %d > %d", len(dataBytes), rs.maxSize)
	}

	meta.Size = int64(len(dataBytes))

	// Marshal metadata
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Store in Redis with TTL
	pipe := rs.client.Pipeline()
	
	ttl := meta.TTL
	if ttl <= 0 {
		ttl = 1 * time.Hour // Default TTL
	}

	pipe.Set(ctx, fullKey+":data", dataBytes, ttl)
	pipe.Set(ctx, fullKey+":meta", metaBytes, ttl)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store in Redis: %w", err)
	}

	return nil
}

// Delete removes a cached entry from Redis
func (rs *RedisStore) Delete(ctx context.Context, key string) error {
	fullKey := rs.prefix + key

	pipe := rs.client.Pipeline()
	pipe.Del(ctx, fullKey+":data")
	pipe.Del(ctx, fullKey+":meta")

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	return nil
}

// PurgePrefix removes all cache entries with keys starting with the prefix
func (rs *RedisStore) PurgePrefix(ctx context.Context, prefix string) error {
	pattern := rs.prefix + prefix + "*"

	// Scan for matching keys
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = rs.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan Redis keys: %w", err)
		}

		// Delete matching keys
		if len(keys) > 0 {
			// Extract unique base keys (without :data or :meta suffix)
			baseKeys := make(map[string]bool)
			for _, key := range keys {
				// Remove prefix and suffix
				baseKey := key
				if len(key) > len(rs.prefix) {
					baseKey = key[len(rs.prefix):]
				}
				// Remove :data or :meta suffix
				if idx := len(baseKey) - 5; idx > 0 && (baseKey[idx:] == ":data" || baseKey[idx:] == ":meta") {
					baseKey = baseKey[:idx]
				}
				baseKeys[baseKey] = true
			}

			// Delete each base key
			for baseKey := range baseKeys {
				rs.Delete(ctx, baseKey)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// Close cleanly shuts down the Redis connection
func (rs *RedisStore) Close() error {
	return rs.client.Close()
}
