package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TaskFunc is the signature for scheduled task handlers.
type TaskFunc func(ctx context.Context) error

// Task represents a scheduled recurring task.
type Task struct {
	Name     string
	Interval time.Duration
	Handler  TaskFunc
}

// Scheduler manages recurring background tasks with distributed locking.
type Scheduler struct {
	queries *sqlcgen.Queries
	tasks   []Task
	stop    chan struct{}
	wg      sync.WaitGroup
	nodeID  string
}

// New creates a new scheduler.
func New(queries *sqlcgen.Queries) *Scheduler {
	return &Scheduler{
		queries: queries,
		stop:    make(chan struct{}),
		nodeID:  uuid.New().String(),
	}
}

// RegisterTask adds a task to the scheduler.
func (s *Scheduler) RegisterTask(name string, interval time.Duration, handler TaskFunc) {
	s.tasks = append(s.tasks, Task{
		Name:     name,
		Interval: interval,
		Handler:  handler,
	})
}

// Start begins running all registered tasks in background goroutines.
func (s *Scheduler) Start(ctx context.Context) {
	for _, task := range s.tasks {
		s.wg.Add(1)
		go s.runTask(ctx, task)
	}
}

// Stop signals all tasks to stop and waits for them to finish.
func (s *Scheduler) Stop() {
	close(s.stop)
	s.wg.Wait()
}

func (s *Scheduler) runTask(ctx context.Context, task Task) {
	defer s.wg.Done()

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	// Run immediately on start, then on each tick
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stop:
			return
		case <-ticker.C:
		}

		acquired, err := s.acquireLock(ctx, task.Name)
		if err != nil {
			slog.Default().Error("failed to acquire scheduler lock", "task", task.Name, "error", err)
			continue
		}
		if !acquired {
			slog.Default().Debug("scheduler lock held by another instance", "task", task.Name)
			continue
		}

		if err := task.Handler(ctx); err != nil {
			slog.Default().Error("scheduler task failed", "task", task.Name, "error", err)
		}

		if err := s.releaseLock(ctx, task.Name); err != nil {
			slog.Default().Error("failed to release scheduler lock", "task", task.Name, "error", err)
		}
	}
}

func (s *Scheduler) acquireLock(ctx context.Context, taskName string) (bool, error) {
	// Try to insert a new lock; if it exists, try to update if expired
	_, err := s.queries.CreateSchedulerLock(ctx, sqlcgen.CreateSchedulerLockParams{
		TaskName:  taskName,
		LockedBy:  s.nodeID,
		ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
	})
	if err == nil {
		return true, nil
	}

	// Lock exists — try to acquire if expired or stale
	lock, err := s.queries.GetSchedulerLock(ctx, taskName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if time.Now().UTC().After(lock.ExpiresAt) {
		_, err := s.queries.UpdateSchedulerLock(ctx, sqlcgen.UpdateSchedulerLockParams{
			TaskName:  taskName,
			LockedBy:  s.nodeID,
			ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		})
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (s *Scheduler) releaseLock(ctx context.Context, taskName string) error {
	return s.queries.DeleteSchedulerLock(ctx, sqlcgen.DeleteSchedulerLockParams{
		TaskName: taskName,
		LockedBy: s.nodeID,
	})
}
