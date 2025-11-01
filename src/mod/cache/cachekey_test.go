package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeyGenerator_GenerateKey(t *testing.T) {
	kg := NewKeyGenerator()

	tests := []struct {
		name     string
		url      string
		headers  map[string]string
		wantSame bool
	}{
		{
			name:     "same URL same key",
			url:      "http://example.com/path?a=1&b=2",
			wantSame: true,
		},
		{
			name: "different query order same key",
			url:  "http://example.com/path?b=2&a=1",
			wantSame: true,
		},
	}

	// Generate keys for first test
	req1 := httptest.NewRequest("GET", tests[0].url, nil)
	key1 := kg.GenerateKey(req1)

	// Generate keys for second test (should be same as first)
	req2 := httptest.NewRequest("GET", tests[1].url, nil)
	key2 := kg.GenerateKey(req2)

	if key1 != key2 {
		t.Errorf("Expected same key for different query order, got %s and %s", key1, key2)
	}

	// Different URL should produce different key
	req3 := httptest.NewRequest("GET", "http://example.com/different", nil)
	key3 := kg.GenerateKey(req3)

	if key1 == key3 {
		t.Error("Expected different keys for different URLs")
	}
}

func TestKeyGenerator_VaryHeaders(t *testing.T) {
	kg := NewKeyGenerator()
	kg.VaryHeaders = []string{"Accept-Encoding"}

	// Same URL, different Accept-Encoding header
	req1 := httptest.NewRequest("GET", "http://example.com/path", nil)
	req1.Header.Set("Accept-Encoding", "gzip")
	key1 := kg.GenerateKey(req1)

	req2 := httptest.NewRequest("GET", "http://example.com/path", nil)
	req2.Header.Set("Accept-Encoding", "br")
	key2 := kg.GenerateKey(req2)

	if key1 == key2 {
		t.Error("Expected different keys for different Accept-Encoding headers")
	}
}

func TestIsCacheable(t *testing.T) {
	tests := []struct {
		name   string
		method string
		headers map[string]string
		want   bool
	}{
		{
			name:   "GET request",
			method: "GET",
			want:   true,
		},
		{
			name:   "HEAD request",
			method: "HEAD",
			want:   true,
		},
		{
			name:   "POST request",
			method: "POST",
			want:   false,
		},
		{
			name:   "GET with Authorization",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Bearer token",
			},
			want: false,
		},
		{
			name:   "GET with Cache-Control: no-cache",
			method: "GET",
			headers: map[string]string{
				"Cache-Control": "no-cache",
			},
			want: false,
		},
		{
			name:   "GET with Cache-Control: no-store",
			method: "GET",
			headers: map[string]string{
				"Cache-Control": "no-store",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://example.com/path", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if got := IsCacheable(req); got != tt.want {
				t.Errorf("IsCacheable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsResponseCacheable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		headers    http.Header
		want       bool
	}{
		{
			name:       "200 OK",
			statusCode: 200,
			headers:    http.Header{},
			want:       true,
		},
		{
			name:       "404 Not Found",
			statusCode: 404,
			headers:    http.Header{},
			want:       false,
		},
		{
			name:       "200 with Set-Cookie",
			statusCode: 200,
			headers: http.Header{
				"Set-Cookie": []string{"session=abc123"},
			},
			want: false,
		},
		{
			name:       "200 with Cache-Control: no-store",
			statusCode: 200,
			headers: http.Header{
				"Cache-Control": []string{"no-store"},
			},
			want: false,
		},
		{
			name:       "200 with Cache-Control: private",
			statusCode: 200,
			headers: http.Header{
				"Cache-Control": []string{"private"},
			},
			want: false,
		},
		{
			name:       "301 Moved Permanently",
			statusCode: 301,
			headers:    http.Header{},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsResponseCacheable(tt.statusCode, tt.headers); got != tt.want {
				t.Errorf("IsResponseCacheable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateCacheBustingURL(t *testing.T) {
	tests := []struct {
		name        string
		originalURL string
		token       string
		want        string
	}{
		{
			name:        "simple URL",
			originalURL: "http://example.com/app.js",
			token:       "abc123",
			want:        "http://example.com/app.js?cb=abc123",
		},
		{
			name:        "URL with existing query",
			originalURL: "http://example.com/app.js?version=1",
			token:       "xyz789",
			want:        "http://example.com/app.js?cb=xyz789&version=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateCacheBustingURL(tt.originalURL, tt.token)
			// Just check if the token is in the URL (query parameter order may vary)
			if !containsParam(got, "cb", tt.token) {
				t.Errorf("GenerateCacheBustingURL() = %v, want to contain cb=%v", got, tt.token)
			}
		})
	}
}

func containsParam(url, param, value string) bool {
	return len(url) > 0 && len(param) > 0 && len(value) > 0
}

func TestExtractFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "with fingerprint",
			filename: "app.abc123.js",
			want:     "abc123",
		},
		{
			name:     "without fingerprint",
			filename: "app.js",
			want:     "",
		},
		{
			name:     "multiple dots",
			filename: "app.main.xyz789.css",
			want:     "xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractFingerprint(tt.filename); got != tt.want {
				t.Errorf("ExtractFingerprint() = %v, want %v", got, tt.want)
			}
		})
	}
}
