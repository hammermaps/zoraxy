package cache

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// VarnishStore provides cache management for Varnish Cache
// Note: This is not a traditional key-value store. Varnish operates as an
// external HTTP cache in front of the application. This implementation
// provides purge/ban capabilities via Varnish's management API.
type VarnishStore struct {
	endpoints  []string // Varnish management endpoints
	httpClient *http.Client
}

// VarnishStoreConfig holds configuration for Varnish store
type VarnishStoreConfig struct {
	Endpoints []string // Varnish management endpoints (e.g., ["http://varnish:6081"])
}

// NewVarnishStore creates a new Varnish cache management interface
func NewVarnishStore(cfg VarnishStoreConfig) (*VarnishStore, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one Varnish endpoint is required")
	}

	return &VarnishStore{
		endpoints: cfg.Endpoints,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Get is not supported for Varnish (it operates as an external cache)
// Returns not found to indicate the application should fetch from origin
func (vs *VarnishStore) Get(ctx context.Context, key string) (io.ReadCloser, *Meta, bool, error) {
	// Varnish handles caching externally, not via this interface
	return nil, nil, false, nil
}

// Put is not supported for Varnish (it caches responses automatically)
// Returns nil to indicate success (Varnish will cache the response itself)
func (vs *VarnishStore) Put(ctx context.Context, key string, body io.Reader, meta *Meta) error {
	// Varnish handles caching automatically when it's in front of the application
	// This is a no-op
	return nil
}

// Delete removes a cached entry from Varnish using PURGE request
func (vs *VarnishStore) Delete(ctx context.Context, key string) error {
	// Send PURGE request to all Varnish endpoints
	// The key should be a URL path
	for _, endpoint := range vs.endpoints {
		url := endpoint + "/" + key
		req, err := http.NewRequestWithContext(ctx, "PURGE", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create PURGE request: %w", err)
		}

		resp, err := vs.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send PURGE request to %s: %w", endpoint, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("PURGE request failed with status %d", resp.StatusCode)
		}
	}

	return nil
}

// PurgePrefix removes all cached entries matching a prefix using BAN request
func (vs *VarnishStore) PurgePrefix(ctx context.Context, prefix string) error {
	// Send BAN request to all Varnish endpoints
	// BAN uses VCL expressions to invalidate multiple objects
	for _, endpoint := range vs.endpoints {
		// Create BAN request with custom header
		req, err := http.NewRequestWithContext(ctx, "BAN", endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create BAN request: %w", err)
		}

		// Set ban expression (matches URLs starting with prefix)
		req.Header.Set("X-Ban-Url", "^"+prefix+".*")

		resp, err := vs.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send BAN request to %s: %w", endpoint, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("BAN request failed with status %d", resp.StatusCode)
		}
	}

	return nil
}

// Ban sends a custom BAN request to Varnish with a specific expression
func (vs *VarnishStore) Ban(ctx context.Context, expression string) error {
	for _, endpoint := range vs.endpoints {
		req, err := http.NewRequestWithContext(ctx, "BAN", endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create BAN request: %w", err)
		}

		// Set custom ban expression
		req.Header.Set("X-Ban-Expression", expression)

		resp, err := vs.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send BAN request to %s: %w", endpoint, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("BAN request failed with status %d", resp.StatusCode)
		}
	}

	return nil
}

// Close cleanly shuts down the Varnish store
func (vs *VarnishStore) Close() error {
	vs.httpClient.CloseIdleConnections()
	return nil
}
