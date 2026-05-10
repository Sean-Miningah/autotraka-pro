package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Event is the canonical envelope used for all system events.
type Event struct {
	Subject       string          `json:"subject"`
	CorrelationID string          `json:"correlation_id"`
	Timestamp     time.Time       `json:"timestamp"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	Payload       json.RawMessage `json:"payload"`
}

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	tenantIDKey      contextKey = "tenant_id"
)

// WithCorrelationID returns a context with the correlation ID set.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// WithTenantID returns a context with the tenant ID set.
func WithTenantID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDKey, id)
}

// HandlerFunc processes a received Event.
type HandlerFunc func(ctx context.Context, event Event) error

// Client provides NATS JetStream publish and subscribe capabilities.
type Client struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	logger *slog.Logger
}

// New creates a Client connected to the given NATS URL and initialises
// the required JetStream streams.
func New(url string, logger *slog.Logger) (*Client, error) {
	if logger == nil {
		logger = slog.Default()
	}

	nc, err := nats.Connect(url,
		nats.Timeout(5*time.Second),
		nats.ReconnectWait(time.Second),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Error("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info("nats reconnected")
		}),
	)
	if err != nil {
		logger.Error("failed to connect to nats", "url", url, "error", err)
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	logger.Info("nats connected", "url", url)

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		logger.Error("failed to initialise jetstream", "error", err)
		return nil, fmt.Errorf("jetstream init: %w", err)
	}

	streams := map[string][]string{
		"MESSAGES":      {"message.>"},
		"CONVERSATIONS": {"conversation.>"},
		"AI":            {"ai.>"},
		"FLOWS":         {"flow.>"},
		"BROADCASTS":    {"broadcast.>"},
	}

	for name, subjects := range streams {
		_, err := js.AddStream(&nats.StreamConfig{
			Name:     name,
			Subjects: subjects,
		})
		if err != nil && err != nats.ErrStreamNameAlreadyInUse {
			nc.Close()
			logger.Error("failed to add jetstream stream", "name", name, "error", err)
			return nil, fmt.Errorf("add stream %s: %w", name, err)
		}
		logger.Info("jetstream stream ready", "name", name, "subjects", subjects)
	}

	return &Client{nc: nc, js: js, logger: logger}, nil
}

// Close drains the NATS connection gracefully.
func (c *Client) Close() error {
	if c.nc == nil {
		return nil
	}
	c.nc.Drain()
	return nil
}

// Publish serialises payload to JSON, wraps it in an Event envelope, and
// publishes to the JetStream subject.
func (c *Client) Publish(ctx context.Context, subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	evt := Event{
		Subject:   subject,
		Timestamp: time.Now().UTC(),
		Payload:   data,
	}

	if cid, ok := ctx.Value(correlationIDKey).(string); ok && cid != "" {
		evt.CorrelationID = cid
	} else {
		evt.CorrelationID = uuid.New().String()
	}
	if tid, ok := ctx.Value(tenantIDKey).(uuid.UUID); ok {
		evt.TenantID = tid
	}

	body, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	_, err = c.js.Publish(subject, body)
	if err != nil {
		return fmt.Errorf("jetstream publish: %w", err)
	}
	return nil
}

// Subscribe creates a durable consumer on subject and invokes handler for
// each received Event.
func (c *Client) Subscribe(ctx context.Context, subject string, handler HandlerFunc) (*nats.Subscription, error) {
	sub, err := c.js.Subscribe(subject, func(msg *nats.Msg) {
		var evt Event
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			msg.Nak()
			return
		}

		msgCtx := context.Background()
		if evt.CorrelationID != "" {
			msgCtx = context.WithValue(msgCtx, correlationIDKey, evt.CorrelationID)
		}
		if evt.TenantID != uuid.Nil {
			msgCtx = context.WithValue(msgCtx, tenantIDKey, evt.TenantID)
		}

		if err := handler(msgCtx, evt); err != nil {
			msg.Nak()
			return
		}
		msg.Ack()
	}, nats.Durable("durable-"+strings.ReplaceAll(subject, ".", "-")), nats.ManualAck(), nats.DeliverNew())
	if err != nil {
		return nil, fmt.Errorf("jetstream subscribe: %w", err)
	}
	return sub, nil
}
