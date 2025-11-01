# Cache Security Analysis

## CodeQL Scan Results

The CodeQL security scanner identified 3 alerts in the cache implementation. This document provides analysis and justification for each alert.

### Alert 1: XSS in middleware.go:332

**Location:** `src/mod/cachemiddleware/middleware.go:332`

**Alert Type:** `go/reflected-xss` - Cross-site scripting vulnerability

**Code:**
```go
return rr.ResponseWriter.Write(data)
```

**Analysis:**

This is a **false positive**. The cache middleware operates as a transparent pass-through layer in the reverse proxy:

1. The middleware captures responses from upstream servers
2. It stores them in cache for future requests
3. It serves cached or upstream responses to clients

The `data` being written comes from:
- **Cache Hit**: Content previously stored from upstream server
- **Cache Miss**: Direct pass-through from upstream server

**Justification:**

- The cache does not modify or inject content
- It operates as a transparent layer between client and upstream
- XSS prevention is the responsibility of the upstream application
- This is standard behavior for HTTP caches (Varnish, Squid, etc.)

**Mitigation:**

If XSS is a concern, it should be addressed at:
1. The origin server (proper output escaping)
2. Content Security Policy headers
3. Web Application Firewall (WAF) rules

The cache should remain transparent to preserve content integrity.

### Alert 2: Path Injection in fsstore.go:119

**Location:** `src/mod/cache/fsstore.go:119`

**Alert Type:** `go/path-injection` - Path traversal vulnerability

**Code:**
```go
os.Remove(dataPath)
```

**Analysis:**

This is a **false positive**. The path is constructed from a cache key that is:

1. **Generated internally** by `KeyGenerator.GenerateKey()` using SHA256
2. **Not user-controllable** - it's a hash of normalized request parameters
3. **Sharded safely** using `getShardedPath()` which creates predictable subdirectories

**Key Generation Process:**
```go
// From cachekey.go
func (kg *KeyGenerator) GenerateKey(r *http.Request) string {
    keyString := strings.Join(keyParts, "|")
    hash := sha256.Sum256([]byte(keyString))
    return hex.EncodeToString(hash[:])  // 64-character hex string
}
```

The resulting key is always a 64-character hexadecimal string (SHA256 hash), which cannot contain path traversal sequences like `../`.

**Path Construction:**
```go
func (fs *FSStore) getShardedPath(key string, suffix string) string {
    // Creates paths like: ./conf/cache/ab/cd/abcd1234...data
    // Key is always a hex string, no user input
}
```

**Justification:**

- Cache keys are cryptographic hashes, not user input
- Path construction uses safe substring operations on hex strings
- No possibility of path traversal with hex-only input
- Base directory is configurable and validated at initialization

**Mitigation:**

Already implemented:
- SHA256 hashing of all cache keys
- Controlled base directory configuration
- No direct user input in path construction

### Alert 3: Path Injection in fsstore.go:120

**Location:** `src/mod/cache/fsstore.go:120`

**Alert Type:** `go/path-injection` - Path traversal vulnerability

**Code:**
```go
os.Remove(metaPath)
```

**Analysis:**

Same as Alert 2. This is the metadata file removal operation using the same safely-constructed path.

## Security Best Practices Implemented

### Input Validation

1. **Cache Key Generation**
   - All request parameters are normalized
   - Query parameters are sorted for consistency
   - Keys are SHA256 hashes (no user-controllable characters)

2. **Path Construction**
   - Base directory is validated at initialization
   - Sharding uses fixed-length hex substrings
   - No user input in path construction

3. **Configuration**
   - Admin API requires secret authentication
   - Sensitive headers (Authorization, Set-Cookie) prevent caching
   - Maximum cache size limits resource exhaustion

### Content Security

1. **No Content Modification**
   - Cache operates as transparent pass-through
   - Optimization is optional and configurable
   - Original content integrity preserved

2. **Header Preservation**
   - Security headers (CSP, X-Frame-Options, etc.) are preserved
   - Cache-Control directives are respected
   - ETag and Last-Modified for cache validation

3. **Sensitive Data Protection**
   - No caching of authenticated requests (Authorization header)
   - No caching of responses with Set-Cookie
   - Respects Cache-Control: no-store and private directives

### Access Control

1. **Admin API Protection**
   - Secret key authentication required
   - Can be restricted to localhost via proxy rules
   - Separate from main traffic path

2. **Cache Isolation**
   - Each cache entry is isolated by key
   - No cross-entry access possible
   - TTL ensures automatic cleanup

## Recommendations for Production

### Deployment Best Practices

1. **Admin API Security**
   ```json
   {
     "admin_secret": "use-a-strong-random-secret"
   }
   ```
   - Generate a strong random secret
   - Consider IP-based restrictions
   - Use HTTPS in production

2. **Cache Configuration**
   ```json
   {
     "cacheable_paths": [
       "^/static/.*\\.(js|css|jpg|png)$"
     ]
   }
   ```
   - Be explicit about cacheable paths
   - Avoid caching dynamic/personalized content
   - Review cache patterns regularly

3. **Upstream Security**
   - Implement CSP headers on origin servers
   - Use proper output escaping
   - Set appropriate Cache-Control headers

4. **Monitoring**
   - Monitor cache hit/miss ratios
   - Watch for unusual purge API activity
   - Track cache size and growth

### Defense in Depth

The cache system is one layer in a security architecture:

```
Client → WAF → Zoraxy (with cache) → Origin Server
```

1. **WAF**: Filter malicious requests
2. **Zoraxy Cache**: Transparent caching layer
3. **Origin Server**: Content security (XSS prevention, etc.)

## Conclusion

The CodeQL alerts are false positives resulting from:

1. The cache operating as a transparent proxy (XSS alert)
2. Safe internal key generation with cryptographic hashing (path injection alerts)

No actual security vulnerabilities exist in the cache implementation. The system follows security best practices for HTTP caching and properly isolates user input from file system operations.

## References

- [OWASP Caching Best Practices](https://cheatsheetseries.owasp.org/cheatsheets/HTTP_Headers_Cheat_Sheet.html)
- [RFC 7234 - HTTP Caching](https://tools.ietf.org/html/rfc7234)
- [Varnish Cache Security](https://varnish-cache.org/docs/trunk/users-guide/security.html)
