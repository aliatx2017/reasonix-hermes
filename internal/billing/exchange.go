// Package billing provides exchange-rate fetching for CNY→USD conversion.
package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"reasonix/internal/netclient"
)

const exchangeRateURL = "https://api.exchangerate-api.com/v4/latest/CNY"

// exchangeRateResponse is the JSON shape returned by exchangerate-api.com.
type exchangeRateResponse struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// DefaultCNYToUSD is the fallback rate used when live fetch fails (≈0.14).
const DefaultCNYToUSD = 0.14

// FetchCNYToUSD fetches the live CNY→USD exchange rate from a free API.
// Returns DefaultCNYToUSD on any error (network timeout, bad response, etc.).
func FetchCNYToUSD() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exchangeRateURL, nil)
	if err != nil {
		return DefaultCNYToUSD
	}
	resp, err := netclient.DefaultClient().Do(req)
	if err != nil {
		return DefaultCNYToUSD
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return DefaultCNYToUSD
	}
	var body exchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return DefaultCNYToUSD
	}
	rate, ok := body.Rates["USD"]
	if !ok || rate <= 0 {
		return DefaultCNYToUSD
	}
	return rate
}
