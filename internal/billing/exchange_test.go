package billing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupExchangeTest swaps in a test server URL and client, and returns a
// cleanup function that restores the originals. Not parallel-safe — tests
// using this helper must not call t.Parallel().
func setupExchangeTest(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	oldURL := exchangeRateTestURL
	oldClient := exchangeRateTestClient
	exchangeRateTestURL = srv.URL
	exchangeRateTestClient = srv.Client()
	t.Cleanup(func() {
		exchangeRateTestURL = oldURL
		exchangeRateTestClient = oldClient
		srv.Close()
	})
	return srv
}

func TestFetchCNYToUSDSuccess(t *testing.T) {
	setupExchangeTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(exchangeRateResponse{
			Base: "CNY",
			Rates: map[string]float64{
				"USD": 0.15,
			},
		})
	})

	rate := FetchCNYToUSD()
	if rate != 0.15 {
		t.Errorf("FetchCNYToUSD = %v, want 0.15", rate)
	}
}

func TestFetchCNYToUSDNoUSDKey(t *testing.T) {
	setupExchangeTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(exchangeRateResponse{
			Base:  "CNY",
			Rates: map[string]float64{}, // no USD
		})
	})

	rate := FetchCNYToUSD()
	if rate != DefaultCNYToUSD {
		t.Errorf("FetchCNYToUSD = %v, want default %v", rate, DefaultCNYToUSD)
	}
}

func TestFetchCNYToUSDZeroRate(t *testing.T) {
	setupExchangeTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(exchangeRateResponse{
			Base: "CNY",
			Rates: map[string]float64{
				"USD": 0, // zero is invalid
			},
		})
	})

	rate := FetchCNYToUSD()
	if rate != DefaultCNYToUSD {
		t.Errorf("FetchCNYToUSD = %v, want default %v", rate, DefaultCNYToUSD)
	}
}

func TestFetchCNYToUSDNegativeRate(t *testing.T) {
	setupExchangeTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(exchangeRateResponse{
			Base: "CNY",
			Rates: map[string]float64{
				"USD": -0.01,
			},
		})
	})

	rate := FetchCNYToUSD()
	if rate != DefaultCNYToUSD {
		t.Errorf("FetchCNYToUSD = %v, want default %v", rate, DefaultCNYToUSD)
	}
}

func TestFetchCNYToUSBogusJSON(t *testing.T) {
	setupExchangeTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not json`))
	})

	rate := FetchCNYToUSD()
	if rate != DefaultCNYToUSD {
		t.Errorf("FetchCNYToUSD = %v, want default %v", rate, DefaultCNYToUSD)
	}
}

func TestFetchCNYToUSDHTTPError(t *testing.T) {
	setupExchangeTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	rate := FetchCNYToUSD()
	if rate != DefaultCNYToUSD {
		t.Errorf("FetchCNYToUSD = %v, want default %v", rate, DefaultCNYToUSD)
	}
}

func TestFetchCNYToUSDNetworkError(t *testing.T) {
	// Not parallel — modifies package-level state.
	oldURL := exchangeRateTestURL
	exchangeRateTestURL = "http://127.0.0.1:1" // nothing listening
	t.Cleanup(func() { exchangeRateTestURL = oldURL })

	rate := FetchCNYToUSD()
	if rate != DefaultCNYToUSD {
		t.Errorf("FetchCNYToUSD = %v, want default %v", rate, DefaultCNYToUSD)
	}
}

func TestDefaultCNYToUSDValue(t *testing.T) {
	t.Parallel()
	if DefaultCNYToUSD <= 0 || DefaultCNYToUSD > 1 {
		t.Errorf("DefaultCNYToUSD = %v, expected 0 < x <= 1", DefaultCNYToUSD)
	}
}
