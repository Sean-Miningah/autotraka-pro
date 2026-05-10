package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	MemberIDKey contextKey = "member_id"
	RoleKey     contextKey = "role"
)

func JWTMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteJSON(w, http.StatusUnauthorized, Envelope{Error: "missing authorization header"})
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				WriteJSON(w, http.StatusUnauthorized, Envelope{Error: "invalid authorization format"})
				return
			}

			token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
				return secret, nil
			})
			if err != nil || !token.Valid {
				WriteJSON(w, http.StatusUnauthorized, Envelope{Error: "invalid or expired token"})
				return
			}

			c, ok := token.Claims.(*claims)
			if !ok {
				WriteJSON(w, http.StatusUnauthorized, Envelope{Error: "invalid token claims"})
				return
			}

			ctx := context.WithValue(r.Context(), TenantIDKey, c.TenantID)
			ctx = context.WithValue(ctx, MemberIDKey, c.MemberID)
			ctx = context.WithValue(ctx, RoleKey, c.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ServiceTokenMiddleware(serviceToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			expected := "Bearer " + serviceToken
			if authHeader != expected {
				WriteJSON(w, http.StatusUnauthorized, Envelope{Error: "invalid service token"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(RoleKey).(string)
		if !ok || role != "admin" {
			WriteJSON(w, http.StatusForbidden, Envelope{Error: "admin access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func WithTenantID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, TenantIDKey, id)
}

func WithMemberID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, MemberIDKey, id)
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, RoleKey, role)
}

func GetTenantID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(TenantIDKey).(uuid.UUID)
	return v
}

func GetMemberID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(MemberIDKey).(uuid.UUID)
	return v
}

func GetRole(ctx context.Context) string {
	v, _ := ctx.Value(RoleKey).(string)
	return v
}