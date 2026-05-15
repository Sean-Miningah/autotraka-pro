package webhook

import (
	"io"
	"net/http"

	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Handler handles incoming platform webhooks.
type Handler struct {
	queries   *sqlcgen.Queries
	eventbus  *eventbus.Client
	channel   channel.Channel
	channelID uuid.UUID
	tenantID  uuid.UUID
}

// NewHandler creates a webhook handler wired to a specific channel.
func NewHandler(queries *sqlcgen.Queries, eb *eventbus.Client, ch channel.Channel, channelID, tenantID uuid.UUID) *Handler {
	return &Handler{
		queries:   queries,
		eventbus:  eb,
		channel:   ch,
		channelID: channelID,
		tenantID:  tenantID,
	}
}

// WhatsApp handles GET (verification) and POST (ingestion) for WhatsApp webhooks.
func (h *Handler) WhatsApp(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleVerification(w, r)
	case http.MethodPost:
		h.handleIngestion(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Instagram handles GET (verification) and POST (ingestion) for Instagram webhooks.
func (h *Handler) Instagram(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleVerification(w, r)
	case http.MethodPost:
		h.handleIngestion(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleVerification(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	verifyToken := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	resp, err := h.channel.VerifyWebhook(mode, verifyToken, challenge)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Write([]byte(resp))
}

func (h *Handler) handleIngestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Hub-Signature-256")
	if err := h.channel.VerifySignature(body, sig); err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	evt, err := h.channel.ParseWebhookEvent(body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Store raw payload in DB (deduplication via unique constraint).
	_, _ = h.queries.CreateWebhookEvent(ctx, sqlcgen.CreateWebhookEventParams{
		TenantID:    h.tenantID,
		ChannelID:   pgtype.UUID{Bytes: h.channelID, Valid: true},
		ChannelType: h.channel.ChannelType(),
		EventID:     evt.EventID,
		RawPayload:  body,
	})

	// Return 200 to Meta immediately, before async processing.
	w.WriteHeader(http.StatusOK)

	// Publish parsed event to NATS asynchronously.
	if h.eventbus != nil {
		ctx = eventbus.WithTenantID(ctx, h.tenantID)
		subject := "message." + h.channel.ChannelType() + ".inbound"
		_ = h.eventbus.Publish(ctx, subject, evt)
	}
}
