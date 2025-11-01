package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// KeyGenerator generates cache keys from HTTP requests
type KeyGenerator struct {
	// IncludeQuery determines whether query parameters are included in the key
	IncludeQuery bool

	// VaryHeaders lists headers to include in cache key generation (e.g., Accept-Encoding)
	VaryHeaders []string

	// CaseSensitive determines if the host and path should be case-sensitive
	CaseSensitive bool
}

// NewKeyGenerator creates a new KeyGenerator with default settings
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{
		IncludeQuery:  true,
		VaryHeaders:   []string{"Accept-Encoding"},
		CaseSensitive: false,
	}
}

// GenerateKey creates a cache key from an HTTP request
func (kg *KeyGenerator) GenerateKey(r *http.Request) string {
	// Build the key components
	var keyParts []string

	// Add scheme
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	keyParts = append(keyParts, scheme)

	// Add host
	host := r.Host
	if !kg.CaseSensitive {
		host = strings.ToLower(host)
	}
	keyParts = append(keyParts, host)

	// Add path
	path := r.URL.Path
	if !kg.CaseSensitive {
		path = strings.ToLower(path)
	}
	keyParts = append(keyParts, path)

	// Add sorted query parameters if enabled
	if kg.IncludeQuery && r.URL.RawQuery != "" {
		query := r.URL.Query()
		keyParts = append(keyParts, kg.normalizeQuery(query))
	}

	// Add vary headers
	if len(kg.VaryHeaders) > 0 {
		for _, header := range kg.VaryHeaders {
			value := r.Header.Get(header)
			if value != "" {
				keyParts = append(keyParts, header+":"+value)
			}
		}
	}

	// Create a hash of the key components
	keyString := strings.Join(keyParts, "|")
	hash := sha256.Sum256([]byte(keyString))
	return hex.EncodeToString(hash[:])
}

// normalizeQuery sorts query parameters for consistent key generation
func (kg *KeyGenerator) normalizeQuery(query url.Values) string {
	if len(query) == 0 {
		return ""
	}

	// Get sorted keys
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build normalized query string
	var parts []string
	for _, k := range keys {
		values := query[k]
		sort.Strings(values)
		for _, v := range values {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	return strings.Join(parts, "&")
}

// GenerateCacheBustingURL adds a cache-busting token to a URL
func GenerateCacheBustingURL(originalURL string, token string) string {
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	query := parsedURL.Query()
	query.Set("cb", token)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String()
}

// ExtractFingerprint extracts a fingerprint from a filename
// e.g., "app.abc123.js" -> "abc123"
func ExtractFingerprint(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) >= 3 {
		// Return the second-to-last part as fingerprint
		return parts[len(parts)-2]
	}
	return ""
}

// IsCacheable determines if a request is cacheable based on method and headers
func IsCacheable(r *http.Request) bool {
	// Only cache GET and HEAD requests
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	// Don't cache requests with Authorization header (unless explicitly allowed)
	if r.Header.Get("Authorization") != "" {
		return false
	}

	// Check Cache-Control: no-cache or no-store
	cacheControl := r.Header.Get("Cache-Control")
	if strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "no-store") {
		return false
	}

	return true
}

// IsResponseCacheable checks if an HTTP response should be cached
func IsResponseCacheable(statusCode int, headers http.Header) bool {
	// Only cache successful responses by default
	if statusCode != http.StatusOK && 
	   statusCode != http.StatusNonAuthoritativeInfo && 
	   statusCode != http.StatusNoContent &&
	   statusCode != http.StatusMovedPermanently &&
	   statusCode != http.StatusFound {
		return false
	}

	// Don't cache responses with Set-Cookie (unless explicitly allowed)
	if headers.Get("Set-Cookie") != "" {
		return false
	}

	// Check Cache-Control directives
	cacheControl := headers.Get("Cache-Control")
	if strings.Contains(cacheControl, "no-store") || strings.Contains(cacheControl, "private") {
		return false
	}

	// Check Pragma: no-cache (HTTP/1.0)
	if headers.Get("Pragma") == "no-cache" {
		return false
	}

	return true
}
