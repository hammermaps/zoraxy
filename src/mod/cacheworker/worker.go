package cacheworker

import (
	"context"
	"log"
	"sync"
	"time"

	"imuslab.com/zoraxy/mod/cache"
	"imuslab.com/zoraxy/mod/cachemiddleware"
	"imuslab.com/zoraxy/mod/optimizer"
)

// Worker processes optimization jobs in the background
type Worker struct {
	queue       chan cachemiddleware.OptimizationJob
	workerCount int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	logger      Logger
}

// Logger interface for worker logging
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// defaultLogger is a simple logger that uses the standard log package
type defaultLogger struct{}

func (dl *defaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (dl *defaultLogger) Println(v ...interface{}) {
	log.Println(v...)
}

// Config holds worker configuration
type Config struct {
	// QueueSize is the size of the job queue
	QueueSize int

	// WorkerCount is the number of concurrent workers
	WorkerCount int

	// RetryAttempts is the number of times to retry failed jobs
	RetryAttempts int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration

	// Logger for worker output
	Logger Logger
}

// DefaultConfig returns default worker configuration
func DefaultConfig() Config {
	return Config{
		QueueSize:     1000,
		WorkerCount:   4,
		RetryAttempts: 3,
		RetryDelay:    5 * time.Second,
		Logger:        &defaultLogger{},
	}
}

// NewWorker creates a new background worker
func NewWorker(config Config) *Worker {
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = 4
	}
	if config.RetryAttempts < 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 5 * time.Second
	}
	if config.Logger == nil {
		config.Logger = &defaultLogger{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &Worker{
		queue:       make(chan cachemiddleware.OptimizationJob, config.QueueSize),
		workerCount: config.WorkerCount,
		ctx:         ctx,
		cancel:      cancel,
		logger:      config.Logger,
	}

	return w
}

// Start starts the worker pool
func (w *Worker) Start() {
	w.logger.Printf("Starting %d cache optimization workers", w.workerCount)

	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.processJobs(i)
	}
}

// Stop stops the worker pool gracefully
func (w *Worker) Stop() {
	w.logger.Println("Stopping cache optimization workers")
	w.cancel()
	close(w.queue)
	w.wg.Wait()
	w.logger.Println("Cache optimization workers stopped")
}

// Enqueue adds a job to the queue (implements JobQueue interface)
func (w *Worker) Enqueue(job cachemiddleware.OptimizationJob) error {
	select {
	case w.queue <- job:
		return nil
	default:
		// Queue is full, drop the job
		w.logger.Println("Optimization queue is full, dropping job for key:", job.Key)
		return nil
	}
}

// processJobs processes jobs from the queue
func (w *Worker) processJobs(workerID int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return

		case job, ok := <-w.queue:
			if !ok {
				return
			}

			// Process the job with retries
			w.processJob(workerID, job)
		}
	}
}

// processJob processes a single optimization job
func (w *Worker) processJob(workerID int, job cachemiddleware.OptimizationJob) {
	// Get the raw content from cache
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reader, meta, found, err := job.Store.Get(ctx, job.Key)
	if err != nil {
		w.logger.Printf("Worker %d: Failed to get cached content for key %s: %v", workerID, job.Key, err)
		return
	}

	if !found {
		w.logger.Printf("Worker %d: Content not found in cache for key %s", workerID, job.Key)
		return
	}
	defer reader.Close()

	// Read content into memory
	var buf []byte
	buf, err = readAll(reader)
	if err != nil {
		w.logger.Printf("Worker %d: Failed to read content for key %s: %v", workerID, job.Key, err)
		return
	}

	// Apply optimization pipeline
	optimized, optimizedMeta, err := job.Pipeline.ApplyToBytes(ctx, buf, meta)
	if err != nil {
		w.logger.Printf("Worker %d: Failed to optimize content for key %s: %v", workerID, job.Key, err)
		return
	}

	// Store optimized content back to cache
	err = job.Store.Put(ctx, job.Key, newBytesReader(optimized), optimizedMeta)
	if err != nil {
		w.logger.Printf("Worker %d: Failed to store optimized content for key %s: %v", workerID, job.Key, err)
		return
	}

	w.logger.Printf("Worker %d: Successfully optimized and cached key %s (original: %d bytes, optimized: %d bytes)",
		workerID, job.Key, len(buf), len(optimized))
}

// readAll reads all data from a reader (helper function)
func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 8192)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return buf, err
		}
	}
	return buf, nil
}

// newBytesReader creates a reader from bytes
func newBytesReader(data []byte) interface {
	Read([]byte) (int, error)
} {
	return &bytesReader{data: data, pos: 0}
}

type bytesReader struct {
	data []byte
	pos  int
}

func (br *bytesReader) Read(p []byte) (n int, err error) {
	if br.pos >= len(br.data) {
		return 0, &eofError{}
	}
	n = copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}

type eofError struct{}

func (e *eofError) Error() string {
	return "EOF"
}

// GetQueueSize returns the current queue size
func (w *Worker) GetQueueSize() int {
	return len(w.queue)
}

// GetQueueCapacity returns the queue capacity
func (w *Worker) GetQueueCapacity() int {
	return cap(w.queue)
}
