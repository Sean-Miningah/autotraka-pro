package contact

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("contact not found")

type Service struct {
	queries *sqlcgen.Queries
}

func NewService(queries *sqlcgen.Queries) *Service {
	return &Service{queries: queries}
}

// PhoneInput is used when creating/updating a contact phone.
type PhoneInput struct {
	Phone string `json:"phone"`
	Label string `json:"label"`
}

// IdentityInput is used when creating/updating a channel identity.
type IdentityInput struct {
	ChannelType     string `json:"channel_type"`
	ChannelIdentity string `json:"channel_identity"`
}

// CreateRequest holds the data for creating a contact.
type CreateRequest struct {
	Name       string          `json:"name"`
	Email      string          `json:"email"`
	Language   string          `json:"language"`
	Phones     []PhoneInput    `json:"phones"`
	Identities []IdentityInput `json:"identities"`
}

// Contact is the enriched contact model with phones and identities.
type Contact struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	Name       string          `json:"name"`
	Email      string          `json:"email"`
	Language   string          `json:"language"`
	Phones     []Phone         `json:"phones"`
	Identities []Identity      `json:"identities"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Phone represents a contact phone number.
type Phone struct {
	ID        uuid.UUID `json:"id"`
	Phone     string    `json:"phone"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

// Identity represents a channel identity linked to a contact.
type Identity struct {
	ID              uuid.UUID `json:"id"`
	ChannelType     string    `json:"channel_type"`
	ChannelIdentity string    `json:"channel_identity"`
	CreatedAt       time.Time `json:"created_at"`
}

// NormalizePhone strips non-digits and ensures E.164 format.
func NormalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if digits == "" {
		return ""
	}
	// If exactly 10 digits, assume US number and prepend country code.
	if len(digits) == 10 {
		digits = "1" + digits
	}
	return "+" + digits
}

func text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func fromText(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

// Create creates a new contact with phones and identities.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*Contact, error) {
	contact, err := s.queries.CreateContact(ctx, sqlcgen.CreateContactParams{
		TenantID: tenantID,
		Name:     text(req.Name),
		Email:    text(req.Email),
		Language: text(req.Language),
	})
	if err != nil {
		return nil, err
	}

	for _, p := range req.Phones {
		_, _ = s.queries.CreateContactPhone(ctx, sqlcgen.CreateContactPhoneParams{
			ContactID: contact.ID,
			Phone:     NormalizePhone(p.Phone),
			Label:     text(p.Label),
		})
	}

	for _, i := range req.Identities {
		_, _ = s.queries.CreateChannelIdentity(ctx, sqlcgen.CreateChannelIdentityParams{
			ContactID:       contact.ID,
			ChannelType:     i.ChannelType,
			ChannelIdentity: i.ChannelIdentity,
		})
	}

	return s.enrich(ctx, contact)
}

// List returns contacts scoped to a tenant with pagination.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]Contact, int64, error) {
	contacts, err := s.queries.ListContactsByTenant(ctx, sqlcgen.ListContactsByTenantParams{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountContactsByTenant(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]Contact, 0, len(contacts))
	for _, c := range contacts {
		enriched, err := s.enrich(ctx, c)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, *enriched)
	}
	return result, count, nil
}

// Get returns a single contact with phones and identities.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*Contact, error) {
	contact, err := s.queries.GetContactByID(ctx, sqlcgen.GetContactByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.enrich(ctx, contact)
}

// UpdateRequest holds optional fields for patching a contact.
type UpdateRequest struct {
	Name     *string `json:"name,omitempty"`
	Email    *string `json:"email,omitempty"`
	Language *string `json:"language,omitempty"`
}

// Update patches a contact's fields.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateRequest) (*Contact, error) {
	existing, err := s.queries.GetContactByID(ctx, sqlcgen.GetContactByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	name := existing.Name
	if req.Name != nil {
		name = text(*req.Name)
	}
	email := existing.Email
	if req.Email != nil {
		email = text(*req.Email)
	}
	lang := existing.Language
	if req.Language != nil {
		lang = text(*req.Language)
	}

	updated, err := s.queries.UpdateContact(ctx, sqlcgen.UpdateContactParams{
		Name:     name,
		Email:    email,
		Language: lang,
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, err
	}
	return s.enrich(ctx, updated)
}

// ResolveRequest holds data for identity resolution.
type ResolveRequest struct {
	ChannelType     string `json:"channel_type"`
	ChannelIdentity string `json:"channel_identity"`
	Name            string `json:"name"`
	Phone           string `json:"phone"`
}

// ResolveOrCreate looks up an existing contact by channel identity or phone,
// and links the identity if found. Otherwise creates a new contact.
func (s *Service) ResolveOrCreate(ctx context.Context, tenantID uuid.UUID, req ResolveRequest) (*Contact, error) {
	// 1. Check by channel identity
	byIdentity, err := s.queries.GetContactByChannelIdentity(ctx, sqlcgen.GetContactByChannelIdentityParams{
		TenantID:        tenantID,
		ChannelType:     req.ChannelType,
		ChannelIdentity: req.ChannelIdentity,
	})
	if err == nil {
		return s.enrich(ctx, byIdentity)
	}

	// 2. Check by normalized phone
	normalized := NormalizePhone(req.Phone)
	if normalized != "" {
		byPhone, err := s.queries.GetContactByPhone(ctx, sqlcgen.GetContactByPhoneParams{
			TenantID: tenantID,
			Phone:    normalized,
		})
		if err == nil {
			// Link new identity to existing contact
			_, _ = s.queries.CreateChannelIdentity(ctx, sqlcgen.CreateChannelIdentityParams{
				ContactID:       byPhone.ID,
				ChannelType:     req.ChannelType,
				ChannelIdentity: req.ChannelIdentity,
			})
			return s.enrich(ctx, byPhone)
		}
	}

	// 3. Create new contact
	return s.Create(ctx, tenantID, CreateRequest{
		Name:     req.Name,
		Phones:   []PhoneInput{{Phone: req.Phone}},
		Identities: []IdentityInput{{ChannelType: req.ChannelType, ChannelIdentity: req.ChannelIdentity}},
	})
}

// Merge consolidates two contacts: moves all phones and identities from source to target, then deletes source.
func (s *Service) Merge(ctx context.Context, tenantID, targetID, sourceID uuid.UUID) (*Contact, error) {
	// Verify both contacts exist and belong to tenant
	_, err := s.queries.GetContactByID(ctx, sqlcgen.GetContactByIDParams{
		ID:       targetID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	_, err = s.queries.GetContactByID(ctx, sqlcgen.GetContactByIDParams{
		ID:       sourceID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Move phones and identities
	_ = s.queries.MoveContactPhones(ctx, sqlcgen.MoveContactPhonesParams{
		ContactID:   targetID,
		ContactID_2: sourceID,
	})
	_ = s.queries.MoveChannelIdentities(ctx, sqlcgen.MoveChannelIdentitiesParams{
		ContactID:   targetID,
		ContactID_2: sourceID,
	})

	// Delete source
	_ = s.queries.DeleteContact(ctx, sqlcgen.DeleteContactParams{
		ID:       sourceID,
		TenantID: tenantID,
	})

	return s.Get(ctx, tenantID, targetID)
}

func (s *Service) enrich(ctx context.Context, row sqlcgen.Contact) (*Contact, error) {
	c := &Contact{
		ID:        row.ID,
		TenantID:  row.TenantID,
		Name:      fromText(row.Name),
		Email:     fromText(row.Email),
		Language:  fromText(row.Language),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	phones, err := s.queries.ListContactPhones(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	c.Phones = make([]Phone, len(phones))
	for i, p := range phones {
		c.Phones[i] = Phone{
			ID:        p.ID,
			Phone:     p.Phone,
			Label:     fromText(p.Label),
			CreatedAt: p.CreatedAt,
		}
	}

	identities, err := s.queries.ListChannelIdentities(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	c.Identities = make([]Identity, len(identities))
	for i, ident := range identities {
		c.Identities[i] = Identity{
			ID:              ident.ID,
			ChannelType:     ident.ChannelType,
			ChannelIdentity: ident.ChannelIdentity,
			CreatedAt:       ident.CreatedAt,
		}
	}

	return c, nil
}
