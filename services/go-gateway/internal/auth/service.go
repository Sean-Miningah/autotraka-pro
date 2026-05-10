package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	queries   *sqlcgen.Queries
	jwtSecret []byte
}

func NewService(queries *sqlcgen.Queries, jwtSecret []byte) *Service {
	return &Service{queries: queries, jwtSecret: jwtSecret}
}

type registerRequest struct {
	TenantName string `json:"tenant_name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

type registerResponse struct {
	TenantID uuid.UUID `json:"tenant_id"`
	MemberID uuid.UUID `json:"member_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
}

type loginRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Password string    `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

var (
	ErrEmailTaken    = errors.New("email already taken")
	ErrInvalidCreds  = errors.New("invalid email or password")
	ErrTokenExpired  = errors.New("refresh token expired")
	ErrTokenNotFound = errors.New("refresh token not found")
)

func (s *Service) Register(ctx context.Context, req registerRequest) (registerResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return registerResponse{}, err
	}

	tenant, err := s.queries.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		Name: req.TenantName,
		Mode: "human_first",
	})
	if err != nil {
		return registerResponse{}, err
	}

	member, err := s.queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		TenantID:     tenant.ID,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "admin",
	})
	if err != nil {
		return registerResponse{}, err
	}

	return registerResponse{
		TenantID: tenant.ID,
		MemberID: member.ID,
		Email:    member.Email,
		Role:     member.Role,
	}, nil
}

func (s *Service) Login(ctx context.Context, req loginRequest) (loginResponse, error) {
	member, err := s.queries.GetMemberByEmail(ctx, sqlcgen.GetMemberByEmailParams{
		TenantID: req.TenantID,
		Email:    req.Email,
	})
	if err != nil {
		return loginResponse{}, ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(member.PasswordHash), []byte(req.Password)); err != nil {
		return loginResponse{}, ErrInvalidCreds
	}

	return s.issueTokens(ctx, member.ID, member.TenantID, member.Role)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (loginResponse, error) {
	tokenHash := sha256Hash(refreshToken)
	stored, err := s.queries.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return loginResponse{}, ErrTokenNotFound
	}

	if time.Now().After(stored.ExpiresAt) {
		s.queries.DeleteRefreshToken(ctx, stored.ID)
		return loginResponse{}, ErrTokenExpired
	}

	member, err := s.queries.GetMemberByID(ctx, stored.MemberID)
	if err != nil {
		return loginResponse{}, err
	}

	s.queries.DeleteRefreshToken(ctx, stored.ID)

	return s.issueTokens(ctx, member.ID, member.TenantID, member.Role)
}

func (s *Service) issueTokens(ctx context.Context, memberID, tenantID uuid.UUID, role string) (loginResponse, error) {
	accessToken, err := s.issueAccessToken(tenantID, memberID, role)
	if err != nil {
		return loginResponse{}, err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return loginResponse{}, err
	}

	refreshHash := sha256Hash(refreshToken)
	_, err = s.queries.CreateRefreshToken(ctx, sqlcgen.CreateRefreshTokenParams{
		MemberID:  memberID,
		TokenHash: refreshHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		return loginResponse{}, err
	}

	return loginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
	}, nil
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sha256Hash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}