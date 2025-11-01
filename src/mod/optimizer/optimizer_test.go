package optimizer

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"imuslab.com/zoraxy/mod/cache"
)

func TestPipeline_Apply(t *testing.T) {
	// Create a simple transform that prefixes content
	prefixTransform := func(prefix string) Transform {
		return func(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error) {
			data, err := io.ReadAll(in)
			if err != nil {
				return nil, nil, err
			}
			result := append([]byte(prefix), data...)
			newMeta := *meta
			newMeta.Size = int64(len(result))
			return io.NopCloser(bytes.NewReader(result)), &newMeta, nil
		}
	}

	pipeline := NewPipeline(
		prefixTransform("A:"),
		prefixTransform("B:"),
	)

	input := "test"
	meta := &cache.Meta{
		ContentType: "text/plain",
		Size:        int64(len(input)),
	}

	result, resultMeta, err := pipeline.Apply(context.Background(), bytes.NewReader([]byte(input)), meta)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	defer result.Close()

	output, err := io.ReadAll(result)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	expected := "B:A:test"
	if string(output) != expected {
		t.Errorf("Expected %s, got %s", expected, string(output))
	}

	if resultMeta.Size != int64(len(expected)) {
		t.Errorf("Expected size %d, got %d", len(expected), resultMeta.Size)
	}
}

func TestPipeline_EmptyPipeline(t *testing.T) {
	pipeline := NewPipeline()

	input := "test"
	meta := &cache.Meta{
		ContentType: "text/plain",
		Size:        int64(len(input)),
	}

	result, resultMeta, err := pipeline.Apply(context.Background(), bytes.NewReader([]byte(input)), meta)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	defer result.Close()

	output, err := io.ReadAll(result)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	if string(output) != input {
		t.Errorf("Expected %s, got %s", input, string(output))
	}

	if resultMeta.Size != meta.Size {
		t.Errorf("Expected size %d, got %d", meta.Size, resultMeta.Size)
	}
}

func TestPipeline_ContextCancellation(t *testing.T) {
	// Create a slow transform
	slowTransform := func(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			data, err := io.ReadAll(in)
			if err != nil {
				return nil, nil, err
			}
			return io.NopCloser(bytes.NewReader(data)), meta, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	pipeline := NewPipeline(slowTransform)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	input := "test"
	meta := &cache.Meta{
		ContentType: "text/plain",
	}

	_, _, err := pipeline.Apply(ctx, bytes.NewReader([]byte(input)), meta)
	if err == nil {
		t.Fatal("Expected context cancellation error")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}
}

func TestPipeline_ApplyToBytes(t *testing.T) {
	uppercaseTransform := func(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error) {
		data, err := io.ReadAll(in)
		if err != nil {
			return nil, nil, err
		}
		result := bytes.ToUpper(data)
		newMeta := *meta
		newMeta.Size = int64(len(result))
		return io.NopCloser(bytes.NewReader(result)), &newMeta, nil
	}

	pipeline := NewPipeline(uppercaseTransform)

	input := []byte("hello world")
	meta := &cache.Meta{
		ContentType: "text/plain",
		Size:        int64(len(input)),
	}

	output, resultMeta, err := pipeline.ApplyToBytes(context.Background(), input, meta)
	if err != nil {
		t.Fatalf("ApplyToBytes failed: %v", err)
	}

	expected := "HELLO WORLD"
	if string(output) != expected {
		t.Errorf("Expected %s, got %s", expected, string(output))
	}

	if resultMeta.Size != int64(len(expected)) {
		t.Errorf("Expected size %d, got %d", len(expected), resultMeta.Size)
	}
}
