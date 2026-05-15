package broadcast

import (
	"context"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

// SchedulerTask picks up scheduled broadcasts and triggers them.
type SchedulerTask struct {
	service *Service
}

// NewSchedulerTask creates a scheduler task for broadcasts.
func NewSchedulerTask(service *Service) *SchedulerTask {
	return &SchedulerTask{service: service}
}

// Run picks up scheduled broadcasts whose time has come and triggers them.
func (t *SchedulerTask) Run(ctx context.Context) error {
	broadcasts, err := t.service.queries.ListScheduledBroadcastsReady(ctx)
	if err != nil {
		return err
	}

	for _, b := range broadcasts {
		// Update status to sending and started_at
		_, _ = t.service.queries.UpdateBroadcastStatus(ctx, sqlcgen.UpdateBroadcastStatusParams{
			Status:      sqlcgen.BroadcastStatusSending,
			StartedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			CompletedAt: pgtype.Timestamptz{},
			ID:          b.ID,
			TenantID:    b.TenantID,
		})

		// Start sending in background
		go t.service.sendBroadcast(context.Background(), b.ID)
	}

	return nil
}
