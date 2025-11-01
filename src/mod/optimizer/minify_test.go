package optimizer

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"imuslab.com/zoraxy/mod/cache"
)

func TestMinifyTransform_HTML(t *testing.T) {
	config := DefaultMinifyConfig()
	transform := MinifyTransform(config)

	input := `
<!DOCTYPE html>
<html>
  <head>
    <title>Test Page</title>
  </head>
  <body>
    <h1>Hello World</h1>
    <p>  This is a test.  </p>
  </body>
</html>
`

	meta := &cache.Meta{
		ContentType: "text/html",
		Size:        int64(len(input)),
	}

	result, resultMeta, err := transform(context.Background(), bytes.NewReader([]byte(input)), meta)
	if err != nil {
		t.Fatalf("MinifyTransform failed: %v", err)
	}
	defer result.Close()

	output, err := io.ReadAll(result)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	// Minified HTML should be shorter
	if len(output) >= len(input) {
		t.Errorf("Expected minified HTML to be shorter: input=%d, output=%d", len(input), len(output))
	}

	// Should still contain the content
	outputStr := string(output)
	if !strings.Contains(outputStr, "Hello World") {
		t.Error("Minified HTML should still contain 'Hello World'")
	}

	if resultMeta.Size != int64(len(output)) {
		t.Errorf("Expected size %d, got %d", len(output), resultMeta.Size)
	}
}

func TestMinifyTransform_CSS(t *testing.T) {
	config := DefaultMinifyConfig()
	transform := MinifyTransform(config)

	input := `
body {
    margin: 0;
    padding: 0;
    background-color: #ffffff;
}

.container {
    width: 100%;
    max-width: 1200px;
}
`

	meta := &cache.Meta{
		ContentType: "text/css",
		Size:        int64(len(input)),
	}

	result, resultMeta, err := transform(context.Background(), bytes.NewReader([]byte(input)), meta)
	if err != nil {
		t.Fatalf("MinifyTransform failed: %v", err)
	}
	defer result.Close()

	output, err := io.ReadAll(result)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	// Minified CSS should be shorter
	if len(output) >= len(input) {
		t.Errorf("Expected minified CSS to be shorter: input=%d, output=%d", len(input), len(output))
	}

	if resultMeta.Size != int64(len(output)) {
		t.Errorf("Expected size %d, got %d", len(output), resultMeta.Size)
	}
}

func TestMinifyTransform_JavaScript(t *testing.T) {
	config := DefaultMinifyConfig()
	transform := MinifyTransform(config)

	input := `
function hello() {
    var message = "Hello, World!";
    console.log(message);
    return message;
}

hello();
`

	meta := &cache.Meta{
		ContentType: "application/javascript",
		Size:        int64(len(input)),
	}

	result, resultMeta, err := transform(context.Background(), bytes.NewReader([]byte(input)), meta)
	if err != nil {
		t.Fatalf("MinifyTransform failed: %v", err)
	}
	defer result.Close()

	output, err := io.ReadAll(result)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	// Minified JS should be shorter
	if len(output) >= len(input) {
		t.Errorf("Expected minified JS to be shorter: input=%d, output=%d", len(input), len(output))
	}

	if resultMeta.Size != int64(len(output)) {
		t.Errorf("Expected size %d, got %d", len(output), resultMeta.Size)
	}
}

func TestMinifyTransform_PassThrough(t *testing.T) {
	config := DefaultMinifyConfig()
	transform := MinifyTransform(config)

	// Test with a content type that shouldn't be minified
	input := "Binary data here"
	meta := &cache.Meta{
		ContentType: "application/octet-stream",
		Size:        int64(len(input)),
	}

	result, resultMeta, err := transform(context.Background(), bytes.NewReader([]byte(input)), meta)
	if err != nil {
		t.Fatalf("MinifyTransform failed: %v", err)
	}
	defer result.Close()

	output, err := io.ReadAll(result)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	// Should pass through unchanged
	if string(output) != input {
		t.Errorf("Expected pass-through, got different output")
	}

	if resultMeta.Size != meta.Size {
		t.Errorf("Expected size %d, got %d", meta.Size, resultMeta.Size)
	}
}

func TestMinifyTransform_NoContentType(t *testing.T) {
	config := DefaultMinifyConfig()
	transform := MinifyTransform(config)

	input := "<html><body>test</body></html>"
	meta := &cache.Meta{
		ContentType: "", // No content type
		Size:        int64(len(input)),
	}

	result, _, err := transform(context.Background(), bytes.NewReader([]byte(input)), meta)
	if err != nil {
		t.Fatalf("MinifyTransform failed: %v", err)
	}
	defer result.Close()

	output, err := io.ReadAll(result)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	// Should pass through without minification
	if string(output) != input {
		t.Errorf("Expected pass-through for empty content type")
	}
}

func TestShouldMinify(t *testing.T) {
	config := DefaultMinifyConfig()

	tests := []struct {
		contentType string
		want        bool
	}{
		{"text/html", true},
		{"text/css", true},
		{"application/javascript", true},
		{"text/javascript", true},
		{"application/json", true},
		{"image/svg+xml", true},
		{"application/xml", false}, // XML disabled in default config
		{"image/png", false},
		{"application/octet-stream", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := shouldMinify(tt.contentType, config)
			if got != tt.want {
				t.Errorf("shouldMinify(%s) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}
