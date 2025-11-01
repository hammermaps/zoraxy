# PageSpeed Integration Adapter

## Overview

This document describes how to integrate external PageSpeed optimization tools with the Zoraxy cache optimizer pipeline. PageSpeed optimization is **disabled by default** and should be explicitly enabled in the configuration.

## Supported Integration Methods

### 1. Local PageSpeed Service

You can run a local PageSpeed optimization service (such as mod_pagespeed or nginx-pagespeed) and integrate it via HTTP calls.

**Setup:**
- Install mod_pagespeed or nginx-pagespeed on a separate server
- Configure the service to accept optimization requests
- Set the PageSpeed endpoint in Zoraxy configuration

**Configuration Example:**
```yaml
cache:
  optimize:
    pagespeed:
      enabled: true
      endpoint: "http://localhost:8081/pagespeed"
      timeout: 30s
```

### 2. Headless Chrome + Lighthouse

Use headless Chrome with Lighthouse for performance analysis and optimization recommendations.

**Setup:**
- Install Chrome/Chromium in headless mode
- Install Lighthouse CLI (`npm install -g lighthouse`)
- Create a wrapper script that runs Lighthouse and extracts recommendations

**Example Wrapper Script:**
```bash
#!/bin/bash
# lighthouse-optimizer.sh
URL=$1
OUTPUT=/tmp/lighthouse-report.json

lighthouse $URL --output=json --output-path=$OUTPUT --only-categories=performance
# Parse JSON and extract optimization recommendations
```

### 3. PageSpeed Insights API (External)

Use Google's PageSpeed Insights API for analysis (rate-limited, requires API key).

**Setup:**
- Obtain a PageSpeed Insights API key from Google Cloud Console
- Configure API key in Zoraxy settings

**Note:** This method is suitable for analysis but not real-time optimization.

## Implementation Guidelines

### Creating a PageSpeed Transform

To add PageSpeed optimization to the pipeline, create a custom transform:

```go
package optimizer

import (
    "context"
    "io"
    "net/http"
    "time"
    
    "imuslab.com/zoraxy/mod/cache"
)

// PageSpeedConfig holds PageSpeed integration configuration
type PageSpeedConfig struct {
    Enabled  bool
    Endpoint string
    Timeout  time.Duration
    APIKey   string
}

// NewPageSpeedTransform creates a transform that calls an external PageSpeed service
func NewPageSpeedTransform(config PageSpeedConfig) Transform {
    return func(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error) {
        if !config.Enabled {
            // Pass through if disabled
            if rc, ok := in.(io.ReadCloser); ok {
                return rc, meta, nil
            }
            return io.NopCloser(in), meta, nil
        }
        
        // Only optimize HTML content
        if !strings.Contains(meta.ContentType, "html") {
            if rc, ok := in.(io.ReadCloser); ok {
                return rc, meta, nil
            }
            return io.NopCloser(in), meta, nil
        }
        
        // Read input
        inputData, err := io.ReadAll(in)
        if err != nil {
            return nil, nil, err
        }
        
        // Call PageSpeed service
        client := &http.Client{Timeout: config.Timeout}
        req, err := http.NewRequestWithContext(ctx, "POST", config.Endpoint, bytes.NewReader(inputData))
        if err != nil {
            // Fall back to original content on error
            return io.NopCloser(bytes.NewReader(inputData)), meta, nil
        }
        
        req.Header.Set("Content-Type", meta.ContentType)
        if config.APIKey != "" {
            req.Header.Set("X-API-Key", config.APIKey)
        }
        
        resp, err := client.Do(req)
        if err != nil {
            // Fall back to original content on error
            return io.NopCloser(bytes.NewReader(inputData)), meta, nil
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != http.StatusOK {
            // Fall back to original content if optimization failed
            return io.NopCloser(bytes.NewReader(inputData)), meta, nil
        }
        
        // Read optimized content
        optimized, err := io.ReadAll(resp.Body)
        if err != nil {
            return io.NopCloser(bytes.NewReader(inputData)), meta, nil
        }
        
        // Update metadata
        newMeta := *meta
        newMeta.Size = int64(len(optimized))
        
        return io.NopCloser(bytes.NewReader(optimized)), &newMeta, nil
    }
}
```

### Usage in Pipeline

Add PageSpeed optimization to the pipeline:

```go
config := PageSpeedConfig{
    Enabled:  true,
    Endpoint: "http://localhost:8081/optimize",
    Timeout:  30 * time.Second,
}

pipeline := NewPipeline(
    MinifyTransform(DefaultMinifyConfig()),
    NewPageSpeedTransform(config),
    BrotliTransform(6),
)
```

## Security Considerations

1. **Network Access**: PageSpeed services should run on localhost or trusted networks
2. **Input Validation**: Validate and sanitize content before sending to external services
3. **Timeouts**: Always set reasonable timeouts to prevent hanging requests
4. **API Keys**: Store API keys securely (not in code or version control)
5. **Rate Limiting**: Implement rate limiting to prevent abuse of external APIs

## License Compatibility

- **mod_pagespeed**: Apache 2.0 License ✓ Compatible with AGPL-3.0
- **Lighthouse**: Apache 2.0 License ✓ Compatible with AGPL-3.0
- **Google PageSpeed Insights API**: Google API Terms (external service, not embedded)

## Recommendations

1. **Start Disabled**: Keep PageSpeed optimization disabled by default
2. **Async Processing**: Use async mode for PageSpeed optimization to avoid blocking requests
3. **Caching**: Cache PageSpeed results to avoid repeated processing
4. **Monitoring**: Monitor PageSpeed service health and failover to non-optimized responses
5. **Testing**: Thoroughly test optimized output before deploying to production

## Example Configuration

```yaml
cache:
  enabled: true
  backend: "fs"
  optimize:
    mode: "async"
    transforms:
      - minify_html: true
      - minify_css: true
      - minify_js: true
      - compress_brotli: true
      - pagespeed: false  # Disabled by default
    pagespeed:
      enabled: false
      endpoint: "http://localhost:8081/optimize"
      timeout: 30s
      api_key: ""  # Set if using PageSpeed Insights API
```

## Future Enhancements

- Support for multiple PageSpeed backends with load balancing
- Automatic fallback if PageSpeed service is unavailable
- Integration with Cloudflare's optimization services
- Support for image optimization via external services
- A/B testing framework for comparing optimized vs non-optimized content
