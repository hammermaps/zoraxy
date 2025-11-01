package optimizer

import (
	"context"
	"io"

	"imuslab.com/zoraxy/mod/cache"
)

// Transform is a function that transforms content from input to output
// It receives the input reader and metadata, and returns a transformed reader and updated metadata
type Transform func(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error)

// Pipeline represents a series of transforms to apply to content
type Pipeline struct {
	transforms []Transform
}

// NewPipeline creates a new optimization pipeline
func NewPipeline(transforms ...Transform) *Pipeline {
	return &Pipeline{
		transforms: transforms,
	}
}

// AddTransform adds a transform to the pipeline
func (p *Pipeline) AddTransform(t Transform) {
	p.transforms = append(p.transforms, t)
}

// Apply applies all transforms in the pipeline sequentially
func (p *Pipeline) Apply(ctx context.Context, in io.Reader, meta *cache.Meta) (io.ReadCloser, *cache.Meta, error) {
	if len(p.transforms) == 0 {
		// No transforms, return input as-is
		if rc, ok := in.(io.ReadCloser); ok {
			return rc, meta, nil
		}
		return io.NopCloser(in), meta, nil
	}

	currentReader := in
	currentMeta := meta

	// Apply each transform in sequence
	for i, transform := range p.transforms {
		// Check context cancellation
		select {
		case <-ctx.Done():
			// Close the current reader if it's a ReadCloser
			if rc, ok := currentReader.(io.ReadCloser); ok {
				rc.Close()
			}
			return nil, nil, ctx.Err()
		default:
		}

		// Apply transform
		nextReader, nextMeta, err := transform(ctx, currentReader, currentMeta)
		if err != nil {
			// Close the current reader on error
			if rc, ok := currentReader.(io.ReadCloser); ok && i > 0 {
				rc.Close()
			}
			return nil, nil, err
		}

		// Close previous reader (except for the original input)
		if i > 0 {
			if rc, ok := currentReader.(io.ReadCloser); ok {
				rc.Close()
			}
		}

		currentReader = nextReader
		currentMeta = nextMeta
	}

	// Ensure we return a ReadCloser
	if rc, ok := currentReader.(io.ReadCloser); ok {
		return rc, currentMeta, nil
	}
	return io.NopCloser(currentReader), currentMeta, nil
}

// ApplyToBytes is a convenience method that applies the pipeline to byte data
func (p *Pipeline) ApplyToBytes(ctx context.Context, data []byte, meta *cache.Meta) ([]byte, *cache.Meta, error) {
	reader := io.NopCloser(io.Reader(readerFromBytes(data)))
	result, resultMeta, err := p.Apply(ctx, reader, meta)
	if err != nil {
		return nil, nil, err
	}
	defer result.Close()

	// Read the result into bytes
	resultBytes, err := io.ReadAll(result)
	if err != nil {
		return nil, nil, err
	}

	return resultBytes, resultMeta, nil
}

type bytesReader struct {
	data []byte
	pos  int
}

func readerFromBytes(data []byte) io.Reader {
	return &bytesReader{data: data, pos: 0}
}

func (br *bytesReader) Read(p []byte) (n int, err error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	n = copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}
