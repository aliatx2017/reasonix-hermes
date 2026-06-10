package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadAPIKey(t *testing.T) {
	t.Setenv("TEST_MCP_KEY", "secret123")
	if got := LoadAPIKey("TEST_MCP_KEY"); got != "secret123" {
		t.Errorf("LoadAPIKey = %q, want %q", got, "secret123")
	}
	if got := LoadAPIKey("NONEXISTENT_KEY"); got != "" {
		t.Errorf("LoadAPIKey for missing key = %q, want empty", got)
	}
	if got := LoadAPIKey(""); got != "" {
		t.Errorf("LoadAPIKey for empty env = %q, want empty", got)
	}
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	mw := &AuthMiddleware{APIKey: "", KeyEnv: "TEST_KEY"}
	called := false
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called when auth is disabled")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_HealthAlwaysPublic(t *testing.T) {
	mw := &AuthMiddleware{APIKey: "secret", KeyEnv: "TEST_KEY"}
	called := false
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("/health should be accessible without auth")
	}
	if w.Code != http.StatusOK {
		t.Errorf("/health status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_NoAuthHeader(t *testing.T) {
	mw := &AuthMiddleware{APIKey: "secret", KeyEnv: "TEST_KEY"}
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth")
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	mw := &AuthMiddleware{APIKey: "secret", KeyEnv: "TEST_KEY"}
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid auth format")
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_WrongKey(t *testing.T) {
	mw := &AuthMiddleware{APIKey: "secret", KeyEnv: "TEST_KEY"}
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with wrong key")
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer wrongkey")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAuthMiddleware_CorrectKey(t *testing.T) {
	mw := &AuthMiddleware{APIKey: "secret", KeyEnv: "TEST_KEY"}
	called := false
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called with correct key")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_Status(t *testing.T) {
	mw1 := &AuthMiddleware{APIKey: "", KeyEnv: "TEST_KEY"}
	if got := mw1.Status(); got != "disabled (no key configured)" {
		t.Errorf("disabled Status = %q", got)
	}

	mw2 := &AuthMiddleware{APIKey: "secret", KeyEnv: "MCP_API_KEY"}
	if got := mw2.Status(); got != "enabled (key set via MCP_API_KEY)" {
		t.Errorf("enabled Status = %q", got)
	}
}