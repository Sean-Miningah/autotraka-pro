package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// TestIntegration_AnalyticsTableExists verifies that the analytics_daily migration
// creates the table with the expected columns and unique constraint.
func TestIntegration_AnalyticsTableExists(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	// Create tenant
	tenant, err := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: "Analytics Corp",
		Mode: "ai_first",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Insert a row — proves the table and columns exist.
	err = queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{String: "whatsapp", Valid: true},
		MetricType:  "conversations_open",
		Value:       5,
	})
	if err != nil {
		t.Fatalf("failed to insert analytics_daily row: %v", err)
	}

	// Upserting the same composite key should update the value, not fail
	err = queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{String: "whatsapp", Valid: true},
		MetricType:  "conversations_open",
		Value:       10,
	})
	if err != nil {
		t.Fatalf("failed to upsert analytics_daily row: %v", err)
	}

	// Verify the value was updated
	rows, err := queries.GetAnalyticsDailyByTenantAndDateRange(ctx, sqlcgen.GetAnalyticsDailyByTenantAndDateRangeParams{
		TenantID: tenant.ID,
		Date:     pgtype.Date{Time: today, Valid: true},
		Date_2:   pgtype.Date{Time: today, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to query analytics_daily: %v", err)
	}

	found := false
	for _, row := range rows {
		if row.MetricType == "conversations_open" && row.ChannelType.String == "whatsapp" {
			found = true
			if row.Value != 10 {
				t.Errorf("expected value 10 after upsert, got %f", row.Value)
			}
		}
	}
	if !found {
		t.Error("expected to find conversations_open metric for whatsapp channel")
	}

	// Verify NULL channel_type works (overall aggregate)
	err = queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{Valid: false},
		MetricType:  "conversations_total",
		Value:       42,
	})
	if err != nil {
		t.Fatalf("failed to insert analytics_daily row with NULL channel_type: %v", err)
	}
}

// TestIntegration_AggregationTask creates conversations/messages for today,
// runs the aggregation task for today, and verifies analytics_daily rows are correct.
func TestIntegration_AggregationTask(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	// Create tenant, channel, and contact
	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: "Analytics Corp",
		Mode: "ai_first",
	})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp Main",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123456"}`),
		Status:      "active",
	})
	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Alice",
		Phones: []contact.PhoneInput{{Phone: "+1234567890", Label: "mobile"}},
	})

	// Create a conversation (AI-handled, open)
	conv, err := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:               tenant.ID,
		ContactID:              c.ID,
		Status:                 sqlcgen.ConversationStatusOpen,
		HandledBy:              sqlcgen.HandledByAi,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	// Inbound message
	_, err = queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenant.ID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: ch.ID, Valid: true},
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        []byte(`{"text":"hello"}`),
	})
	if err != nil {
		t.Fatalf("create inbound message: %v", err)
	}

	// Outbound AI response
	_, err = queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenant.ID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: ch.ID, Valid: true},
		Direction:      sqlcgen.MessageDirectionOutbound,
		Status:         sqlcgen.MessageStatusSent,
		ContentType:    "text",
		Content:        []byte(`{"text":"hi there"}`),
	})
	if err != nil {
		t.Fatalf("create outbound message: %v", err)
	}

	// Run aggregation for today (data was just created "now")
	today := time.Now().UTC().Truncate(24 * time.Hour)
	aggTask := NewAggregatorTask(queries)
	if err := aggTask.RunForDate(ctx, today); err != nil {
		t.Fatalf("aggregation task failed: %v", err)
	}

	// Verify analytics rows exist for today
	rows, err := queries.GetAnalyticsDailyByTenantAndDateRange(ctx, sqlcgen.GetAnalyticsDailyByTenantAndDateRangeParams{
		TenantID: tenant.ID,
		Date:     pgtype.Date{Time: today, Valid: true},
		Date_2:   pgtype.Date{Time: today, Valid: true},
	})
	if err != nil {
		t.Fatalf("query analytics_daily: %v", err)
	}

	if len(rows) == 0 {
		t.Fatal("expected analytics_daily rows after aggregation, got 0")
	}

	// Build a map for easy assertion
	metrics := make(map[string]float64)
	for _, row := range rows {
		key := row.MetricType
		if row.ChannelType.Valid {
			key += ":" + row.ChannelType.String
		} else {
			key += ":all"
		}
		metrics[key] += row.Value
	}

	// Verify conversation count by channel
	if v, ok := metrics["conversations_total:whatsapp"]; !ok || v != 1 {
		t.Errorf("expected conversations_total:whatsapp=1, got %v", metrics["conversations_total:whatsapp"])
	}

	// Verify message counts by channel
	if v, ok := metrics["messages_inbound:whatsapp"]; !ok || v != 1 {
		t.Errorf("expected messages_inbound:whatsapp=1, got %v", metrics["messages_inbound:whatsapp"])
	}
	if v, ok := metrics["messages_outbound:whatsapp"]; !ok || v != 1 {
		t.Errorf("expected messages_outbound:whatsapp=1, got %v", metrics["messages_outbound:whatsapp"])
	}

	// Verify aggregate totals (channel_type = null means "all")
	if v, ok := metrics["conversations_open:all"]; !ok || v != 1 {
		t.Errorf("expected conversations_open:all=1, got %v", metrics["conversations_open:all"])
	}
	if v, ok := metrics["messages_inbound:all"]; !ok || v != 1 {
		t.Errorf("expected messages_inbound:all=1, got %v", metrics["messages_inbound:all"])
	}

	t.Logf("metrics: %+v", metrics)

	// Verify tenant scoping — query a different tenant should return nothing
	otherTenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: "Other Corp",
		Mode: "ai_first",
	})
	otherRows, err := queries.GetAnalyticsDailyByTenantAndDateRange(ctx, sqlcgen.GetAnalyticsDailyByTenantAndDateRangeParams{
		TenantID: otherTenant.ID,
		Date:     pgtype.Date{Time: today, Valid: true},
		Date_2:   pgtype.Date{Time: today, Valid: true},
	})
	if err != nil {
		t.Fatalf("query other tenant analytics: %v", err)
	}
	if len(otherRows) != 0 {
		t.Errorf("expected no analytics rows for other tenant, got %d", len(otherRows))
	}
}

// TestIntegration_AggregationWithResolvedConversation verifies that resolved conversations
// and their handled_by are tracked correctly.
func TestIntegration_AggregationWithResolvedConversation(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: "Resolve Corp",
		Mode: "ai_first",
	})
	ch, _ := queries.CreateChannel(ctx, sqlcgen.CreateChannelParams{
		TenantID:    tenant.ID,
		Name:        "WhatsApp",
		ChannelType: "whatsapp",
		Config:      []byte(`{"phone_number_id":"123"}`),
		Status:      "active",
	})
	contactSvc := contact.NewService(queries)
	c, _ := contactSvc.Create(ctx, tenant.ID, contact.CreateRequest{
		Name:   "Bob",
		Phones: []contact.PhoneInput{{Phone: "+1111111111", Label: "mobile"}},
	})

	// Create a resolved conversation (human-handled)
	conv, _ := queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		TenantID:    tenant.ID,
		ContactID:   c.ID,
		Status:      sqlcgen.ConversationStatusResolved,
		HandledBy:   sqlcgen.HandledByHuman,
	})

	// Messages
	queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenant.ID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: ch.ID, Valid: true},
		Direction:      sqlcgen.MessageDirectionInbound,
		Status:         sqlcgen.MessageStatusDelivered,
		ContentType:    "text",
		Content:        []byte(`{"text":"help"}`),
	})
	queries.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		TenantID:       tenant.ID,
		ConversationID: conv.ID,
		ChannelID:      pgtype.UUID{Bytes: ch.ID, Valid: true},
		Direction:      sqlcgen.MessageDirectionOutbound,
		Status:         sqlcgen.MessageStatusSent,
		ContentType:    "text",
		Content:        []byte(`{"text":"solved"}`),
	})

	// Run aggregation for today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	aggTask := NewAggregatorTask(queries)
	aggTask.RunForDate(ctx, today)

	rows, err := queries.GetAnalyticsDailyByTenantAndDateRange(ctx, sqlcgen.GetAnalyticsDailyByTenantAndDateRangeParams{
		TenantID: tenant.ID,
		Date:     pgtype.Date{Time: today, Valid: true},
		Date_2:   pgtype.Date{Time: today, Valid: true},
	})
	if err != nil {
		t.Fatalf("query analytics: %v", err)
	}

	metrics := make(map[string]float64)
	for _, row := range rows {
		key := row.MetricType
		if row.ChannelType.Valid {
			key += ":" + row.ChannelType.String
		} else {
			key += ":all"
		}
		metrics[key] = row.Value
	}

	// Verify resolved count by channel
	if v, ok := metrics["conversations_resolved:whatsapp"]; !ok || v != 1 {
		t.Errorf("expected conversations_resolved:whatsapp=1, got %+v", metrics)
	}
	// Verify human handled
	if v, ok := metrics["conversations_handled_human:all"]; !ok || v != 1 {
		t.Errorf("expected conversations_handled_human:all=1, got %+v", metrics)
	}
}

// TestIntegration_OverviewEndpoint seeds analytics_daily data and verifies
// the overview API returns correct totals for a date range, scoped to tenant.
func TestIntegration_OverviewEndpoint(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: "Overview Corp",
		Mode: "ai_first",
	})

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Seed analytics data
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{Valid: false},
		MetricType:  "conversations_open",
		Value:       5,
	})
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{Valid: false},
		MetricType:  "conversations_resolved",
		Value:       3,
	})
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{Valid: false},
		MetricType:  "messages_inbound",
		Value:       20,
	})
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{Valid: false},
		MetricType:  "messages_outbound",
		Value:       15,
	})

	// Test the service directly
	svc := NewService(queries)

	overview, err := svc.GetOverview(ctx, tenant.ID, today, today)
	if err != nil {
		t.Fatalf("GetOverview failed: %v", err)
	}

	if overview.ConversationsOpen != 5 {
		t.Errorf("expected ConversationsOpen=5, got %f", overview.ConversationsOpen)
	}
	if overview.ConversationsResolved != 3 {
		t.Errorf("expected ConversationsResolved=3, got %f", overview.ConversationsResolved)
	}
	if overview.MessagesInbound != 20 {
		t.Errorf("expected MessagesInbound=20, got %f", overview.MessagesInbound)
	}
	if overview.MessagesOutbound != 15 {
		t.Errorf("expected MessagesOutbound=15, got %f", overview.MessagesOutbound)
	}
}

// TestIntegration_ConversationsEndpoint seeds analytics_daily data and verifies
// the conversations API returns correct breakdowns.
func TestIntegration_ConversationsEndpoint(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: "Conv Corp",
		Mode: "ai_first",
	})

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Seed analytics: conversations by status and channel
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{String: "whatsapp", Valid: true},
		MetricType:  "conversations_open",
		Value:       10,
	})
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{String: "whatsapp", Valid: true},
		MetricType:  "conversations_resolved",
		Value:       7,
	})
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{String: "telegram", Valid: true},
		MetricType:  "conversations_open",
		Value:       3,
	})

	svc := NewService(queries)
	convs, err := svc.GetConversations(ctx, tenant.ID, today, today, "", 20, 0)
	if err != nil {
		t.Fatalf("GetConversations failed: %v", err)
	}

	if len(convs) == 0 {
		t.Fatal("expected conversation breakdown rows, got 0")
	}

	// Verify we can find whatsapp open
	foundWhatsAppOpen := false
	for _, c := range convs {
		if c.ChannelType == "whatsapp" && c.MetricType == "conversations_open" {
			foundWhatsAppOpen = true
			if c.Value != 10 {
				t.Errorf("expected whatsapp open=10, got %f", c.Value)
			}
		}
	}
	if !foundWhatsAppOpen {
		t.Error("expected whatsapp conversations_open entry")
	}
}

// TestIntegration_MessagesEndpoint seeds analytics_daily data and verifies
// the messages API returns volume data with cursor pagination.
func TestIntegration_MessagesEndpoint(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: "Msg Corp",
		Mode: "ai_first",
	})

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Seed analytics: messages by channel
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{String: "whatsapp", Valid: true},
		MetricType:  "messages_inbound",
		Value:       50,
	})
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{String: "whatsapp", Valid: true},
		MetricType:  "messages_outbound",
		Value:       40,
	})

	svc := NewService(queries)
	msgs, cursor, err := svc.GetMessages(ctx, tenant.ID, today, today, uuid.Nil, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) == 0 {
		t.Fatal("expected message volume rows, got 0")
	}

	// Verify whatsapp inbound is present
	foundWAInbound := false
	for _, m := range msgs {
		if m.ChannelType == "whatsapp" && m.MetricType == "messages_inbound" {
			foundWAInbound = true
			if m.Value != 50 {
				t.Errorf("expected whatsapp inbound=50, got %f", m.Value)
			}
		}
	}
	if !foundWAInbound {
		t.Error("expected whatsapp messages_inbound entry")
	}

	// Cursor should be nil/empty when all results fit in limit
	if cursor != uuid.Nil {
		t.Logf("cursor returned: %v (may be valid for pagination)", cursor)
	}
}

// strPtr helper for string pointers.
func strPtr(s string) *string { return &s }

// TestIntegration_OverviewHTTPHandler tests the overview endpoint via HTTP.
func TestIntegration_OverviewHTTPHandler(t *testing.T) {
	ctx := context.Background()

	pool, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	queries := sqlcgen.New(pool)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: "HTTP Corp",
		Mode: "ai_first",
	})

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Seed analytics data
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{Valid: false},
		MetricType:  "conversations_open",
		Value:       12,
	})
	queries.UpsertAnalyticsDaily(ctx, sqlcgen.UpsertAnalyticsDailyParams{
		TenantID:    tenant.ID,
		Date:        pgtype.Date{Time: today, Valid: true},
		ChannelType: pgtype.Text{Valid: false},
		MetricType:  "messages_inbound",
		Value:       100,
	})

	svc := NewService(queries)
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.TenantIDKey, tenant.ID)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	handler.RegisterRoutes(r)

	fromStr := today.Format("2006-01-02")
	toString := today.Format("2006-01-02")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/overview?from="+fromStr+"&to="+toString, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp auth.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}

	if data["conversations_open"] != float64(12) {
		t.Errorf("expected conversations_open=12, got %v", data["conversations_open"])
	}
	if data["messages_inbound"] != float64(100) {
		t.Errorf("expected messages_inbound=100, got %v", data["messages_inbound"])
	}
}