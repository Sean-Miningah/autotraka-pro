package contact

import (
	"context"
	"errors"
	"testing"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/testutil"
)

func TestCreateContactWithPhonesAndIdentities(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	// Create tenant
	tenant, err := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	contact, err := svc.Create(ctx, tenant.ID, CreateRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Language: "en",
		Phones: []PhoneInput{
			{Phone: "+1 (555) 123-4567", Label: "primary"},
			{Phone: "+1-555-765-4321", Label: "work"},
		},
		Identities: []IdentityInput{
			{ChannelType: "whatsapp", ChannelIdentity: "15551234567"},
		},
	})
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}

	if contact.Name != "John Doe" {
		t.Errorf("expected name John Doe, got %s", contact.Name)
	}
	if contact.Email != "john@example.com" {
		t.Errorf("expected email john@example.com, got %s", contact.Email)
	}
	if len(contact.Phones) != 2 {
		t.Fatalf("expected 2 phones, got %d", len(contact.Phones))
	}
	if contact.Phones[0].Phone != "+15551234567" {
		t.Errorf("expected normalized phone +15551234567, got %s", contact.Phones[0].Phone)
	}
	if len(contact.Identities) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(contact.Identities))
	}
	if contact.Identities[0].ChannelType != "whatsapp" {
		t.Errorf("expected channel type whatsapp, got %s", contact.Identities[0].ChannelType)
	}
}

func TestCreateContactNormalizesPhone(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})

	cases := []struct {
		input    string
		expected string
	}{
		{"+1 (555) 123-4567", "+15551234567"},
		{"1-555-123-4567", "+15551234567"},
		{"555-123-4567", "+15551234567"},
		{"+15551234567", "+15551234567"},
	}

	for _, tc := range cases {
		contact, err := svc.Create(ctx, tenant.ID, CreateRequest{
			Name:   "Test",
			Phones: []PhoneInput{{Phone: tc.input}},
		})
		if err != nil {
			t.Fatalf("input %q: %v", tc.input, err)
		}
		if contact.Phones[0].Phone != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, contact.Phones[0].Phone)
		}
	}
}

func TestListContactsScopedToTenant(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenantA, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "A", Mode: "human_first"})
	tenantB, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "B", Mode: "human_first"})

	svc.Create(ctx, tenantA.ID, CreateRequest{Name: "Contact A1"})
	svc.Create(ctx, tenantA.ID, CreateRequest{Name: "Contact A2"})
	svc.Create(ctx, tenantB.ID, CreateRequest{Name: "Contact B1"})

	contacts, count, err := svc.List(ctx, tenantA.ID, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts, got %d", len(contacts))
	}
}

func TestGetContactWithPhonesAndIdentities(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})
	created, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name:     "John",
		Phones:   []PhoneInput{{Phone: "+15551234567"}},
		Identities: []IdentityInput{{ChannelType: "whatsapp", ChannelIdentity: "wa123"}},
	})

	contact, err := svc.Get(ctx, tenant.ID, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if contact.Name != "John" {
		t.Errorf("expected name John, got %s", contact.Name)
	}
	if len(contact.Phones) != 1 {
		t.Errorf("expected 1 phone, got %d", len(contact.Phones))
	}
	if len(contact.Identities) != 1 {
		t.Errorf("expected 1 identity, got %d", len(contact.Identities))
	}
}

func TestGetContactWrongTenant(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenantA, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "A", Mode: "human_first"})
	tenantB, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "B", Mode: "human_first"})
	created, _ := svc.Create(ctx, tenantA.ID, CreateRequest{Name: "John"})

	_, err := svc.Get(ctx, tenantB.ID, created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateContact(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})
	created, _ := svc.Create(ctx, tenant.ID, CreateRequest{Name: "John", Email: "john@old.com"})

	updated, err := svc.Update(ctx, tenant.ID, created.ID, UpdateRequest{
		Name:  strPtr("Jane"),
		Email: strPtr("jane@new.com"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Jane" {
		t.Errorf("expected name Jane, got %s", updated.Name)
	}
	if updated.Email != "jane@new.com" {
		t.Errorf("expected email jane@new.com, got %s", updated.Email)
	}
}

func TestResolveOrCreateAutoMergeByPhone(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})
	existing, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name:   "Existing",
		Phones: []PhoneInput{{Phone: "+15551234567"}},
	})

	// Resolve with same phone via new identity — should link to existing contact
	resolved, err := svc.ResolveOrCreate(ctx, tenant.ID, ResolveRequest{
		ChannelType:     "whatsapp",
		ChannelIdentity: "15551234567",
		Name:            "New Name",
		Phone:           "+1 (555) 123-4567",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.ID != existing.ID {
		t.Errorf("expected same contact ID, got %v vs %v", resolved.ID, existing.ID)
	}

	// Identity should now be linked to existing contact
	contact, _ := svc.Get(ctx, tenant.ID, existing.ID)
	if len(contact.Identities) != 1 {
		t.Errorf("expected 1 identity on existing contact, got %d", len(contact.Identities))
	}
}

func TestResolveOrCreateCreatesNewWhenNoMatch(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})

	resolved, err := svc.ResolveOrCreate(ctx, tenant.ID, ResolveRequest{
		ChannelType:     "whatsapp",
		ChannelIdentity: "9999999999",
		Name:            "New Contact",
		Phone:           "+19999999999",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Name != "New Contact" {
		t.Errorf("expected name New Contact, got %s", resolved.Name)
	}
	if len(resolved.Phones) != 1 {
		t.Errorf("expected 1 phone, got %d", len(resolved.Phones))
	}
	if len(resolved.Identities) != 1 {
		t.Errorf("expected 1 identity, got %d", len(resolved.Identities))
	}
}

func TestMergeContacts(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenant, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "Acme", Mode: "human_first"})
	target, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name:   "Target",
		Phones: []PhoneInput{{Phone: "+11111111111"}},
	})
	source, _ := svc.Create(ctx, tenant.ID, CreateRequest{
		Name:     "Source",
		Phones:   []PhoneInput{{Phone: "+12222222222"}},
		Identities: []IdentityInput{{ChannelType: "whatsapp", ChannelIdentity: "wa_source"}},
	})

	merged, err := svc.Merge(ctx, tenant.ID, target.ID, source.ID)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if merged.ID != target.ID {
		t.Errorf("expected merged contact to be target")
	}

	// Target should now have both phones and the identity
	contact, _ := svc.Get(ctx, tenant.ID, target.ID)
	if len(contact.Phones) != 2 {
		t.Errorf("expected 2 phones after merge, got %d", len(contact.Phones))
	}
	if len(contact.Identities) != 1 {
		t.Errorf("expected 1 identity after merge, got %d", len(contact.Identities))
	}

	// Source should be gone
	_, err = svc.Get(ctx, tenant.ID, source.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected source to be deleted, got %v", err)
	}
}

func TestMergeContactsWrongTenant(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	queries := sqlcgen.New(pool)
	svc := NewService(queries)

	tenantA, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "A", Mode: "human_first"})
	tenantB, _ := queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{Name: "B", Mode: "human_first"})
	target, _ := svc.Create(ctx, tenantA.ID, CreateRequest{Name: "Target"})
	source, _ := svc.Create(ctx, tenantB.ID, CreateRequest{Name: "Source"})

	_, err := svc.Merge(ctx, tenantA.ID, target.ID, source.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-tenant merge, got %v", err)
	}
}

func strPtr(s string) *string {
	return &s
}
