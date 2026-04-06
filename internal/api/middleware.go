package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"simple-orderbook/internal/core/ports"
)

type contextKey string

const UserIDKey contextKey = "userID"

func RateLimitMiddleware(limiter ports.RateLimiter, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			allowed, err := limiter.Allow(r.Context(), key)
			if err != nil {
				slog.Error("rate limiter check failed", "error", err, "key", key)
			} else if !allowed {
				WriteError(w, "rate-limit-exceeded", "Too many requests", "You have exceeded your request quota", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func IPKey(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				WriteError(w, "missing-token", "Unauthorized", "Authentication token is missing", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				slog.Warn("invalid jwt attempt", "error", err, "ip", IPKey(r))
				WriteError(w, "invalid-token", "Unauthorized", "Your session has expired or is invalid", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				WriteError(w, "invalid-subject", "Unauthorized", "Token subject is not a valid UUID", http.StatusUnauthorized)
				return
			}

			sub, ok := claims["sub"].(string)
			userID, err := uuid.Parse(sub)
			if err != nil {
				WriteError(w, "invalid-subject", "Unauthorized", "Token subject is not a valid UUID", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
