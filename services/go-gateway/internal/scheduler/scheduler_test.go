package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
)

func TestTaskRunsAtInterval(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	var counter int32
	s := New(queries)
	s.RegisterTask("test-counter", 100*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})
	s.Start(ctx)

	// Wait for task to run at least twice
	time.Sleep(350 * time.Millisecond)
	s.Stop()

	if atomic.LoadInt32(&counter) < 2 {
		t.Errorf("expected task to run at least 2 times, got %d", counter)
	}
}

func TestDistributedLockPreventsDuplicateRuns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	var counter int32

	// Create two schedulers (simulating two replicas)
	s1 := New(queries)
	s1.nodeID = "node-1"
	s1.RegisterTask("shared-task", 100*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	s2 := New(queries)
	s2.nodeID = "node-2"
	s2.RegisterTask("shared-task", 100*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	s1.Start(ctx)
	s2.Start(ctx)

	time.Sleep(350 * time.Millisecond)
	s1.Stop()
	s2.Stop()

	// With distributed locking, only one node should run the task
	// So counter should be roughly equal to the number of intervals, not doubled
	c := atomic.LoadInt32(&counter)
	if c > 5 {
		t.Errorf("expected distributed lock to prevent duplicate runs, got %d executions", c)
	}
}

func TestLockExpiresAndIsReacquired(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	var counter int32
	s := New(queries)
	s.RegisterTask("expiring-task", 100*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	// Manually insert an expired lock
	_, _ = queries.CreateSchedulerLock(ctx, sqlcgen.CreateSchedulerLockParams{
		TaskName:  "expiring-task",
		LockedBy:  "old-node",
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
	})

	s.Start(ctx)
	time.Sleep(200 * time.Millisecond)
	s.Stop()

	if atomic.LoadInt32(&counter) == 0 {
		t.Error("expected task to run after acquiring expired lock")
	}
}

func TestHealthCheckerUpdatesChannelStatus(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)

	// Create tenant and channel
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Test", Mode: "human_first"})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WA",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123","access_token":"token","app_secret":"secret"}`),
		Status:      "active",
	})

	hc := NewHealthChecker(queries)
	if err := hc.CheckAllChannels(ctx); err != nil {
		t.Fatalf("CheckAllChannels failed: %v", err)
	}

	// Verify health status was updated
	result, err := queries.GetChannelHealth(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetChannelHealth failed: %v", err)
	}
	if result.HealthStatus.String != "healthy" {
		t.Errorf("expected health_status healthy, got %s", result.HealthStatus.String)
	}
	if !result.HealthCheckedAt.Valid {
		t.Error("expected health_checked_at to be set")
	}
}
