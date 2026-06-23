package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- symbol ---

func TestSymbolCNY(t *testing.T) {
	t.Parallel()
	if got := symbol("CNY"); got != "¥" {
		t.Errorf("symbol(CNY) = %q", got)
	}
}

func TestSymbolRMB(t *testing.T) {
	t.Parallel()
	if got := symbol("RMB"); got != "¥" {
		t.Errorf("symbol(RMB) = %q", got)
	}
}

func TestSymbolUSD(t *testing.T) {
	t.Parallel()
	if got := symbol("USD"); got != "$" {
		t.Errorf("symbol(USD) = %q", got)
	}
}

func TestSymbolUnknown(t *testing.T) {
	t.Parallel()
	if got := symbol("EUR"); got != "EUR " {
		t.Errorf("symbol(EUR) = %q, want \"EUR \"", got)
	}
}

func TestSymbolEmpty(t *testing.T) {
	t.Parallel()
	if got := symbol(""); got != "" {
		t.Errorf("symbol(\"\") = %q, want empty", got)
	}
}

func TestSymbolLowercase(t *testing.T) {
	t.Parallel()
	// symbol should be case-insensitive.
	if got := symbol("usd"); got != "$" {
		t.Errorf("symbol(usd) = %q", got)
	}
}

// --- Display ---

func TestDisplayNil(t *testing.T) {
	t.Parallel()
	var b *Balance
	if got := b.Display(); got != "" {
		t.Errorf("nil Display = %q", got)
	}
}

func TestDisplayEmptyInfos(t *testing.T) {
	t.Parallel()
	b := &Balance{Available: true}
	if got := b.Display(); got != "" {
		t.Errorf("empty infos Display = %q", got)
	}
}

func TestDisplayPrefersCNY(t *testing.T) {
	t.Parallel()
	b := &Balance{Infos: []Info{
		{Currency: "USD", TotalBalance: "10.00"},
		{Currency: "CNY", TotalBalance: "50.00"},
	}}
	if got := b.Display(); got != "¥50.00" {
		t.Errorf("Display = %q, want ¥50.00", got)
	}
}

func TestDisplayFallsBackToFirst(t *testing.T) {
	t.Parallel()
	b := &Balance{Infos: []Info{
		{Currency: "EUR", TotalBalance: "25.00"},
	}}
	if got := b.Display(); got != "EUR 25.00" {
		t.Errorf("Display = %q, want \"EUR 25.00\"", got)
	}
}

// --- Fetch edge cases ---

func TestFetchContextCancelled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := FetchWithClient(ctx, nil, srv.URL, "key")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestFetchMalformedJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()
	_, err := FetchWithClient(context.Background(), nil, srv.URL, "key")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error should mention decode: %v", err)
	}
}

func TestFetchNoAPIKey(t *testing.T) {
	t.Parallel()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[]}`))
	}))
	defer srv.Close()
	_, err := FetchWithClient(context.Background(), nil, srv.URL, "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("Authorization should be empty when no key, got %q", gotAuth)
	}
}

func TestFetchWhitespaceURL(t *testing.T) {
	t.Parallel()
	b, err := FetchWithClient(context.Background(), nil, "   ", "key")
	if err != nil || b != nil {
		t.Fatalf("whitespace URL should return (nil,nil), got (%v, %v)", b, err)
	}
}

func TestFetchServerUnavailable(t *testing.T) {
	t.Parallel()
	// Use a URL that won't connect.
	_, err := FetchWithClient(context.Background(), nil, "http://127.0.0.1:1", "key")
	if err == nil {
		t.Fatal("expected error for unavailable server")
	}
}
