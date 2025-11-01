package optimizer

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
	"imuslab.com/zoraxy/mod/cache"
)

// CompressionType represents the type of compression
type CompressionType string

const (
	CompressionGzip   CompressionType = "gzip"
	CompressionBrotli CompressionType = "br"
	CompressionNone   CompressionType = ""
)

// CompressConfig holds configuration for compression
type CompressConfig struct {
	// Type specifies the compression algorithm to use
	Type CompressionType

	// Level specifies the compression level (1-9 for gzip, 0-11 for brotli)
	Level int

	// MinSize is the minimum size (in bytes) before compression is applied
	MinSize int64
}

// DefaultGzipConfig returns the default gzip compression configuration
func DefaultGzipConfig() CompressConfig {
	return CompressConfig{
		Type:    CompressionGzip,
		Level:   gzip.DefaultCompression,
		MinSize: 1024, // 1KB minimum
	}
}

// DefaultBrotliConfig returns the default brotli compression configuration
func DefaultBrotliConfig() CompressConfig {
	return CompressConfig{
		Type:    CompressionBrotli,
		Level:   6, // Default brotli level
		MinSize: 1024,
	}
}

// CompressTransform creates a Transform that compresses content
func CompressTransform(config CompressConfig) Transform {
	return func(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error) {
		// Skip compression if already compressed
		if meta.Encoding != "" && meta.Encoding != "identity" {
			if rc, ok := in.(io.ReadCloser); ok {
				return rc, meta, nil
			}
			return io.NopCloser(in), meta, nil
		}

		// Read input into buffer
		var buf bytes.Buffer
		written, err := io.Copy(&buf, in)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read input: %w", err)
		}

		// Skip compression if below minimum size
		if written < config.MinSize {
			newMeta := *meta
			newMeta.Size = written
			return io.NopCloser(&buf), &newMeta, nil
		}

		// Compress based on type
		var compressed bytes.Buffer
		var encoding string

		switch config.Type {
		case CompressionGzip:
			w, err := gzip.NewWriterLevel(&compressed, config.Level)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create gzip writer: %w", err)
			}
			if _, err := io.Copy(w, &buf); err != nil {
				w.Close()
				return nil, nil, fmt.Errorf("failed to compress with gzip: %w", err)
			}
			w.Close()
			encoding = "gzip"

		case CompressionBrotli:
			w := brotli.NewWriterLevel(&compressed, config.Level)
			if _, err := io.Copy(w, &buf); err != nil {
				w.Close()
				return nil, nil, fmt.Errorf("failed to compress with brotli: %w", err)
			}
			w.Close()
			encoding = "br"

		default:
			// No compression
			newMeta := *meta
			newMeta.Size = written
			return io.NopCloser(&buf), &newMeta, nil
		}

		// Check if compression actually reduced size
		if int64(compressed.Len()) >= written {
			// Compression didn't help, return uncompressed
			newMeta := *meta
			newMeta.Size = written
			return io.NopCloser(&buf), &newMeta, nil
		}

		// Update metadata
		newMeta := *meta
		newMeta.Encoding = encoding
		newMeta.Size = int64(compressed.Len())

		return io.NopCloser(&compressed), &newMeta, nil
	}
}

// GzipTransform creates a Transform that compresses with gzip
func GzipTransform(level int) Transform {
	return CompressTransform(CompressConfig{
		Type:    CompressionGzip,
		Level:   level,
		MinSize: 1024,
	})
}

// BrotliTransform creates a Transform that compresses with brotli
func BrotliTransform(level int) Transform {
	return CompressTransform(CompressConfig{
		Type:    CompressionBrotli,
		Level:   level,
		MinSize: 1024,
	})
}

// DecompressTransform creates a Transform that decompresses content
func DecompressTransform() Transform {
	return func(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error) {
		if meta.Encoding == "" || meta.Encoding == "identity" {
			// Not compressed
			if rc, ok := in.(io.ReadCloser); ok {
				return rc, meta, nil
			}
			return io.NopCloser(in), meta, nil
		}

		var decompressed io.Reader
		var err error

		switch meta.Encoding {
		case "gzip":
			decompressed, err = gzip.NewReader(in)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
			}

		case "br":
			decompressed = brotli.NewReader(in)

		default:
			// Unknown encoding, pass through
			if rc, ok := in.(io.ReadCloser); ok {
				return rc, meta, nil
			}
			return io.NopCloser(in), meta, nil
		}

		// Read decompressed content
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, decompressed); err != nil {
			return nil, nil, fmt.Errorf("failed to decompress: %w", err)
		}

		// Close gzip reader if applicable
		if closer, ok := decompressed.(io.Closer); ok {
			closer.Close()
		}

		// Update metadata
		newMeta := *meta
		newMeta.Encoding = ""
		newMeta.Size = int64(buf.Len())

		return io.NopCloser(&buf), &newMeta, nil
	}
}

// IsCompressible checks if a content type is typically compressible
func IsCompressible(contentType string) bool {
	compressible := []string{
		"text/",
		"application/json",
		"application/javascript",
		"application/xml",
		"application/x-javascript",
		"application/xhtml+xml",
		"image/svg+xml",
	}

	for _, prefix := range compressible {
		if len(contentType) >= len(prefix) && contentType[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}
