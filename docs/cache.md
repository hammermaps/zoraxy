# Zoraxy Cache Plugin

## Overview

The Zoraxy cache plugin provides a modular response caching system with support for multiple storage backends (Filesystem, Redis, Varnish), optional content optimization (minification, compression), and administrative purge APIs.

## Architecture

### Components

1. **Cache Stores** (`mod/cache/`)
   - `CacheStore` interface: Defines standard operations (Get, Put, Delete, PurgePrefix)
   - `FSStore`: Filesystem-based cache with sharded directory structure
   - `RedisStore`: Redis-based cache with TTL support
   - `VarnishStore`: Varnish integration via PURGE/BAN HTTP methods

2. **Optimizer** (`mod/optimizer/`)
   - `Transform` interface: Pipeline for content transformations
   - `MinifyTransform`: HTML/CSS/JS minification using tdewolff/minify
   - `CompressTransform`: Brotli and gzip compression support
   - `Pipeline`: Chains multiple transforms together

3. **Cache Middleware** (`mod/cachemiddleware/`)
   - `Middleware`: HTTP middleware that intercepts requests
   - `AdminHandler`: Provides REST API for cache management
   - Statistics tracking (hits, misses, bypasses, errors)

4. **Worker** (`mod/cacheworker/`)
   - Background job queue for async optimization
   - Configurable worker pool
   - Retry logic with exponential backoff

## Configuration

### Example Configuration

Create `conf/cache_conf.json`:

```json
{
  "enabled": true,
  "backend": "fs",
  "fs": {
    "root": "./conf/cache",
    "shard_depth": 2
  },
  "redis": {
    "addr": "localhost:6379",
    "password": "",
    "db": 0
  },
  "varnish": {
    "endpoints": ["http://varnish:6081"]
  },
  "ttl": 3600,
  "max_cache_size": 104857600,
  "optimize": {
    "mode": "sync",
    "minify_css": true,
    "minify_js": true,
    "minify_html": true,
    "compress_brotli": true,
    "compress_gzip": false
  },
  "cacheable_paths": [
    "^/static/.*\\.(js|css|jpg|jpeg|png|gif|svg|ico|woff|woff2|ttf|eot)$"
  ],
  "admin_secret": "your-secret-key-here"
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | false | Enable/disable caching |
| `backend` | string | "fs" | Backend type: "fs", "redis", or "varnish" |
| `ttl` | int | 3600 | Default time-to-live in seconds |
| `max_cache_size` | int64 | 104857600 | Maximum cache entry size in bytes (100MB) |
| `optimize.mode` | string | "disabled" | Optimization mode: "sync", "async", or "disabled" |
| `cacheable_paths` | []string | - | Regex patterns for cacheable paths |
| `admin_secret` | string | "" | Secret key for admin API access |

## Cache Key Generation

Cache keys are generated using SHA256 hash of:
- Request scheme (http/https)
- Host
- Path
- Sorted query parameters (if enabled)
- Vary headers (e.g., Accept-Encoding)

### Example:

```
Request: https://example.com/api/data?page=1&limit=10
Cache Key: sha256("https|example.com|/api/data|limit=10&page=1|Accept-Encoding:gzip")
```

## Cacheability Rules

### Request Criteria

A request is cacheable if:
- Method is GET or HEAD
- No `Authorization` header (unless explicitly allowed)
- No `Cache-Control: no-cache` or `no-store` in request headers

### Response Criteria

A response is cacheable if:
- Status code is 200, 203, 204, 301, or 302
- No `Set-Cookie` header (unless explicitly allowed)
- No `Cache-Control: no-store` or `private` in response headers
- No `Pragma: no-cache` header (HTTP/1.0 compatibility)

## Optimization Pipeline

### Minification

Supported content types:
- **HTML**: `text/html`
- **CSS**: `text/css`
- **JavaScript**: `text/javascript`, `application/javascript`, `application/x-javascript`
- **JSON**: `application/json`
- **SVG**: `image/svg+xml`

### Compression

Two compression algorithms are supported:

1. **Brotli** (recommended)
   - Better compression ratios
   - Slower compression (suitable for cache write-once scenarios)
   - Encoding: `br`

2. **Gzip**
   - Wider compatibility
   - Faster compression
   - Encoding: `gzip`

### Optimization Modes

#### Sync Mode
- Optimizes content before caching and serving
- Adds latency to cache miss requests
- Suitable for low-traffic sites or static assets

#### Async Mode
- Caches raw response immediately
- Serves raw content on first request
- Optimizes in background
- Subsequent requests get optimized content
- Suitable for high-traffic sites

#### Disabled
- No optimization performed
- Lowest overhead

## Admin API

### Authentication

All admin endpoints require authentication via:
1. Bearer token in `Authorization` header: `Authorization: Bearer <secret>`
2. Query parameter: `?secret=<secret>`

### Endpoints

#### Purge Single Entry

```http
POST /_cache/purge
Content-Type: application/json

{
  "key": "sha256-hash-of-cache-key"
}
```

Response:
```json
{
  "success": true,
  "message": "Cache entry purged successfully",
  "key": "sha256-hash-of-cache-key"
}
```

#### Purge by Prefix

```http
POST /_cache/purge-prefix
Content-Type: application/json

{
  "prefix": "/static/js/"
}
```

Response:
```json
{
  "success": true,
  "message": "Cache entries purged successfully",
  "prefix": "/static/js/"
}
```

#### Cache Status

```http
GET /_cache/status?secret=your-secret-key
```

Response:
```json
{
  "enabled": true,
  "backend": "filesystem",
  "stats": {
    "hits": 1234,
    "misses": 567,
    "puts": 567,
    "errors": 0,
    "bypasses": 890,
    "hit_rate": 68.5
  },
  "config": {
    "optimization_mode": "sync",
    "default_ttl": "1h0m0s",
    "max_cache_size": 104857600
  }
}
```

#### Varnish BAN (Varnish backend only)

```http
POST /_cache/ban
Content-Type: application/json

{
  "expression": "req.url ~ ^/api/",
  "prefix": "/api/"
}
```

## Storage Backends

### Filesystem

**Advantages:**
- No external dependencies
- Simple to set up
- Good for small to medium deployments

**Configuration:**
```json
{
  "backend": "fs",
  "fs": {
    "root": "./conf/cache",
    "shard_depth": 2
  }
}
```

**Sharding:**
- Distributes cache files across subdirectories
- Prevents single directory from having too many files
- Example: Key `abcd1234` with depth 2 → `./conf/cache/ab/cd/abcd1234.data`

### Redis

**Advantages:**
- Automatic TTL management
- High performance
- Distributed caching support
- Memory-efficient

**Configuration:**
```json
{
  "backend": "redis",
  "redis": {
    "addr": "localhost:6379",
    "password": "",
    "db": 0
  }
}
```

**Key Structure:**
- Data: `zoraxy:cache:<key>:data`
- Metadata: `zoraxy:cache:<key>:meta`

### Varnish

**Advantages:**
- Industry-standard HTTP cache
- Highly optimized
- Advanced invalidation rules
- Load balancing integration

**Configuration:**
```json
{
  "backend": "varnish",
  "varnish": {
    "endpoints": [
      "http://varnish1:6081",
      "http://varnish2:6081"
    ]
  }
}
```

**Note:** Varnish operates as an external cache in front of Zoraxy. The VarnishStore provides purge/BAN capabilities via HTTP management API.

## Response Headers

### Cache Hit

```
X-Cache: HIT
Age: 123
Cache-Control: public, max-age=3477
ETag: "abc123"
Content-Encoding: br
```

### Cache Miss

```
X-Cache: MISS
Cache-Control: public, max-age=3600
```

## Integration with Zoraxy

### Step 1: Load Configuration

During startup, Zoraxy loads `conf/cache_conf.json` and initializes the cache system.

### Step 2: Wrap Proxy Handler

The cache middleware wraps the reverse proxy handler:

```go
// In start.go
func startupSequence() {
    // ... existing initialization ...
    
    // Initialize cache system
    if err := initCacheSystem(); err != nil {
        SystemWideLogger.Printf("Failed to initialize cache: %v", err)
    }
}

// In reverseproxy.go (example integration)
func ReverseProxyInit() {
    // ... create dynamicProxyRouter ...
    
    // Wrap with cache middleware
    cachedHandler := wrapHandlerWithCache(dynamicProxyRouter.mux)
    dynamicProxyRouter.mux = cachedHandler
    
    // Register cache admin APIs
    registerCacheAPIs(webminPanelMux)
}
```

### Step 3: Shutdown

During shutdown, the cache system is cleanly closed:

```go
func ShutdownSeq() {
    // ... existing shutdown ...
    
    shutdownCacheSystem()
}
```

## Performance Considerations

### Memory Usage

- **Filesystem**: Minimal memory overhead (only active requests)
- **Redis**: Memory proportional to cache size
- **Async optimization**: Additional memory for job queue

### Disk I/O

- **Filesystem**: Uses atomic writes (temp file + rename)
- **Sharding**: Reduces directory lookup overhead
- **Metadata**: Stored separately to avoid parsing content

### Network

- **Redis**: Additional network latency (~1-2ms local)
- **Varnish**: Depends on Varnish deployment topology

## Monitoring

### Metrics to Track

1. **Hit Rate**: `hits / (hits + misses)`
2. **Miss Rate**: `misses / (hits + misses)`
3. **Bypass Rate**: `bypasses / total_requests`
4. **Error Rate**: `errors / total_requests`
5. **Cache Size**: Total bytes cached
6. **Optimization Queue**: Async jobs pending

### Logging

Cache operations are logged to Zoraxy's system logger:

```
[INFO] Cache system initialized (TTL: 3600s, Max size: 104857600 bytes)
[INFO] Cache middleware enabled
[INFO] Cache worker started with 4 workers
[INFO] Cache HIT: /static/app.js (Age: 123s)
[INFO] Cache MISS: /api/data (caching new entry)
[WARN] Cache entry exceeds maximum size: 110000000 > 104857600
[ERROR] Failed to store in cache: disk full
```

## Security

### Admin API Protection

- Always use a strong `admin_secret`
- Consider IP-based restrictions via access rules
- Use HTTPS in production

### Cache Poisoning Prevention

- Caching is disabled for authenticated requests
- Vary header support ensures correct content per client
- No caching of Set-Cookie responses

### DoS Protection

- Maximum cache entry size prevents memory exhaustion
- Queue limits prevent worker overflow
- TTL ensures automatic cleanup

## Troubleshooting

### Cache Not Working

1. Check `enabled: true` in configuration
2. Verify cacheable path patterns match your URLs
3. Check request method (only GET/HEAD)
4. Verify no `Authorization` headers
5. Check response headers (no `Set-Cookie`, no `Cache-Control: private`)

### Low Hit Rate

1. Review `cacheable_paths` patterns
2. Check `TTL` value (too low?)
3. Verify `Vary` headers are correctly configured
4. Monitor cache evictions (size limits?)

### High Memory Usage

1. Reduce `max_cache_size`
2. Lower `TTL` to expire entries sooner
3. Use Redis backend instead of filesystem
4. Disable async optimization if queue is growing

### Optimization Not Working

1. Check `optimize.mode` is not "disabled"
2. Verify content types are supported
3. Check optimizer errors in logs
4. Ensure worker is running (async mode)

## License Compatibility

All dependencies are compatible with AGPL-3.0:

- **tdewolff/minify**: MIT License ✓
- **andybalholm/brotli**: MIT License ✓
- **go-redis/redis**: BSD-2-Clause License ✓

## Future Enhancements

- [ ] Prometheus metrics export
- [ ] Cache warming/preloading
- [ ] Stale-while-revalidate support
- [ ] Multi-backend write-through
- [ ] Image optimization via external services
- [ ] Distributed cache invalidation (Redis Pub/Sub)
- [ ] A/B testing framework
- [ ] Advanced Vary header handling

## Examples

### Basic Static Asset Caching

```json
{
  "enabled": true,
  "backend": "fs",
  "ttl": 86400,
  "cacheable_paths": [
    "^/static/.*",
    "^/assets/.*"
  ],
  "optimize": {
    "mode": "sync",
    "minify_css": true,
    "minify_js": true,
    "compress_brotli": true
  }
}
```

### API Response Caching with Redis

```json
{
  "enabled": true,
  "backend": "redis",
  "redis": {
    "addr": "redis:6379",
    "password": "secret",
    "db": 0
  },
  "ttl": 300,
  "cacheable_paths": [
    "^/api/v1/.*"
  ],
  "optimize": {
    "mode": "disabled"
  }
}
```

### High-Traffic Site with Async Optimization

```json
{
  "enabled": true,
  "backend": "redis",
  "ttl": 3600,
  "max_cache_size": 10485760,
  "cacheable_paths": [
    "^/.*\\.(html|css|js)$"
  ],
  "optimize": {
    "mode": "async",
    "minify_html": true,
    "minify_css": true,
    "minify_js": true,
    "compress_brotli": true
  }
}
```

## Support

For issues, questions, or contributions, please visit:
https://github.com/tobychui/zoraxy

## Credits

Developed as part of the Zoraxy project.
Licensed under AGPL-3.0.
