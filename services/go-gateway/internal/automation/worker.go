package automation

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
)

// WorkerPool polls for and resumes paused/waiting automation runs.
type WorkerPool struct {
	queries  *sqlcgen.Queries
	engine   *Engine
	logger   *slog.Logger
	limit    int32
	interval time.Duration
	workers  int
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewWorkerPool creates a worker pool for resuming automation runs.
func NewWorkerPool(queries *sqlcgen.Queries, engine *Engine, logger *slog.Logger, workers int) *WorkerPool {
	if logger == nil {
		logger = slog.Default()
	}
	if workers <= 0 {
		workers = 4
	}
	return &WorkerPool{
		queries:  queries,
		engine:   engine,
		logger:   logger,
		limit:    int32(workers * 2),
		interval: 5 * time.Second,
		workers:  workers,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the worker pool polling loop.
func (w *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)
		go w.worker(ctx)
	}
}

// Stop signals the worker pool to shut down.
func (w *WorkerPool) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}

func (w *WorkerPool) worker(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.pollAndResume(ctx)
		}
	}
}

func (w *WorkerPool) pollAndResume(ctx context.Context) {
	runs, err := w.queries.PollAutomationRunsForResume(ctx, w.limit)
	if err != nil {
		w.logger.Error("failed to poll automation runs", "error", err)
		return
	}

	for _, run := range runs {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
		}

		w.logger.Info("resuming automation run", "run_id", run.ID, "status", run.Status, "resume_at", run.ResumeAt)
		if err := w.engine.ResumeRun(ctx, run.ID); err != nil {
			w.logger.Error("failed to resume run", "run_id", run.ID, "error", err)
		}
	}
}
