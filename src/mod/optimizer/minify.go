package optimizer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
	"imuslab.com/zoraxy/mod/cache"
)

// MinifyConfig holds configuration for minification
type MinifyConfig struct {
	HTML bool
	CSS  bool
	JS   bool
	JSON bool
	SVG  bool
	XML  bool
}

// DefaultMinifyConfig returns the default minification configuration
func DefaultMinifyConfig() MinifyConfig {
	return MinifyConfig{
		HTML: true,
		CSS:  true,
		JS:   true,
		JSON: true,
		SVG:  true,
		XML:  false, // XML minification can be risky for some applications
	}
}

// NewMinifier creates a minifier with the specified configuration
func NewMinifier(config MinifyConfig) *minify.M {
	m := minify.New()

	if config.HTML {
		m.AddFunc("text/html", html.Minify)
	}

	if config.CSS {
		m.AddFunc("text/css", css.Minify)
	}

	if config.JS {
		m.AddFunc("text/javascript", js.Minify)
		m.AddFunc("application/javascript", js.Minify)
		m.AddFunc("application/x-javascript", js.Minify)
	}

	if config.JSON {
		m.AddFunc("application/json", json.Minify)
	}

	if config.SVG {
		m.AddFunc("image/svg+xml", svg.Minify)
	}

	if config.XML {
		m.AddFunc("application/xml", xml.Minify)
		m.AddFunc("text/xml", xml.Minify)
	}

	return m
}

// MinifyTransform creates a Transform that minifies content based on content type
func MinifyTransform(config MinifyConfig) Transform {
	minifier := NewMinifier(config)

	return func(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error) {
		// Check if this content type should be minified
		contentType := meta.ContentType
		if contentType == "" {
			// No content type, pass through
			if rc, ok := in.(io.ReadCloser); ok {
				return rc, meta, nil
			}
			return io.NopCloser(in), meta, nil
		}

		// Extract media type (ignore charset and other parameters)
		mediaType := contentType
		if idx := strings.IndexByte(contentType, ';'); idx != -1 {
			mediaType = strings.TrimSpace(contentType[:idx])
		}

		// Check if minifier handles this type
		if !shouldMinify(mediaType, config) {
			if rc, ok := in.(io.ReadCloser); ok {
				return rc, meta, nil
			}
			return io.NopCloser(in), meta, nil
		}

		// Read input into buffer
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, in); err != nil {
			return nil, nil, fmt.Errorf("failed to read input: %w", err)
		}

		// Minify
		var minified bytes.Buffer
		if err := minifier.Minify(mediaType, &minified, &buf); err != nil {
			// If minification fails, return original content
			if rc, ok := in.(io.ReadCloser); ok {
				return rc, meta, nil
			}
			return io.NopCloser(&buf), meta, nil
		}

		// Update metadata
		newMeta := *meta
		newMeta.Size = int64(minified.Len())

		return io.NopCloser(&minified), &newMeta, nil
	}
}

// shouldMinify checks if a content type should be minified
func shouldMinify(contentType string, config MinifyConfig) bool {
	ct := strings.ToLower(contentType)

	// Check HTML - be specific to avoid false matches
	if config.HTML && (ct == "text/html" || strings.HasPrefix(ct, "text/html;")) {
		return true
	}

	// Check CSS - be specific to avoid false matches
	if config.CSS && (ct == "text/css" || strings.HasPrefix(ct, "text/css;")) {
		return true
	}

	// Check JavaScript - check common variations
	if config.JS && (ct == "text/javascript" || ct == "application/javascript" ||
		ct == "application/x-javascript" || strings.HasPrefix(ct, "text/javascript;") ||
		strings.HasPrefix(ct, "application/javascript;") ||
		strings.HasPrefix(ct, "application/x-javascript;")) {
		return true
	}

	// Check JSON - be specific with standard types
	if config.JSON && (ct == "application/json" || strings.HasPrefix(ct, "application/json;")) {
		return true
	}

	// Check SVG - exact match only
	if config.SVG && (ct == "image/svg+xml" || strings.HasPrefix(ct, "image/svg+xml;")) {
		return true
	}

	// Check XML - exact matches only
	if config.XML && (ct == "application/xml" || ct == "text/xml" ||
		strings.HasPrefix(ct, "application/xml;") || strings.HasPrefix(ct, "text/xml;")) {
		return true
	}

	return false
}
