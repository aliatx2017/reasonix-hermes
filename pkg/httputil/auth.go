// Package httputil provides HTTP middleware for MCP servers.
package httputil

import (
	"crypto/subtle"
	"net/http"
	"os"
)

// AuthMiddleware checks for a Bearer token in the Authorization header.
// If APIKey is empty, all requests are allowed (auth disabled).
// If APIKey is set, requests must include `Authorization: Bearer <key>`.
// The health endpoint (/health) is always unauthenticated.
type AuthMiddleware struct {
	// APIKey is the expected bearer token. If empty, auth is disabled.
	APIKey string
	// KeyEnv is the environment variable name checked at startup.
	// Used for doctor/diagnostic output, not for runtime checks.
	KeyEnv string
}

// LoadAPIKey reads the API key from the environment variable.
// Returns empty string (auth disabled) if the env var is not set.
func LoadAPIKey(envVar string) string {
	if envVar == "" {
		return ""
	}
	return os.Getenv(envVar)
}

// Wrap returns an http.Handler that enforces Bearer auth on all paths
// except /health, then delegates to next.
func (a *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health endpoint is always public
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Auth disabled — no key configured
		if a.APIKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"unauthorized","message":"Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		// Expect "Bearer <token>"
		const prefix = "Bearer "
		if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
			http.Error(w, `{"error":"unauthorized","message":"Invalid authorization format, expected Bearer token"}`, http.StatusUnauthorized)
			return
		}

		token := authHeader[len(prefix):]
		if subtle.ConstantTimeCompare([]byte(token), []byte(a.APIKey)) != 1 {
			http.Error(w, `{"error":"forbidden","message":"Invalid API key"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Status returns a human-readable auth status for diagnostics.
func (a *AuthMiddleware) Status() string {
	if a.APIKey == "" {
		return "disabled (no key configured)"
	}
	return "enabled (key set via " + a.KeyEnv + ")"
}