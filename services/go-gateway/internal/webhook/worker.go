package webhook

import (
	"context"
	"log/slog"
	"time"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
)

// Worker polls unprocessed webhook events and publishes them to NATS.
type Worker struct {
	queries  *sqlcgen.Queries
	eventbus *eventbus.Client
	channel  channel.Channel
	interval time.Duration
	stop     chan struct{}
}

// NewWorker creates a background worker for retrying unprocessed webhook events.
func NewWorker(queries *sqlcgen.Queries, eb *eventbus.Client, ch channel.Channel) *Worker {
	return &Worker{
		queries:  queries,
		eventbus: eb,
		channel:  ch,
		interval: 5 * time.Second,
		stop:     make(chan struct{}),
	}
}

// Run starts the worker polling loop.
func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	w.interval = interval
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// Stop signals the worker to stop.
func (w *Worker) Stop() {
	close(w.stop)
}

func (w *Worker) processBatch(ctx context.Context) {
	events, err := w.queries.ListUnprocessedWebhookEvents(ctx, 100)
	if err != nil {
		slog.Default().Error("failed to list unprocessed events", "error", err)
		return
	}

	for _, evt := range events {
		if err := w.queries.MarkWebhookEventProcessed(ctx, evt.ID); err != nil {
			slog.Default().Error("failed to mark event processed", "event_id", evt.EventID, "error", err)
			continue
		}

		if w.eventbus != nil {
			msgCtx := eventbus.WithTenantID(ctx, evt.TenantID)
			_ = w.eventbus.Publish(msgCtx, "message.whatsapp.inbound", map[string]string{
				"event_id": evt.EventID,
				"payload":  string(evt.RawPayload),
			})
		}
	}
}
