package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type claims struct {
	TenantID uuid.UUID `json:"tenant_id"`
	MemberID uuid.UUID `json:"member_id"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

func (s *Service) issueAccessToken(tenantID, memberID uuid.UUID, role string) (string, error) {
	c := claims{
		TenantID: tenantID,
		MemberID: memberID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(s.jwtSecret)
}