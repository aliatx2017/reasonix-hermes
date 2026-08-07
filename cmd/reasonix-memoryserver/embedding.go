// embedding.go — dense embedding support for the Hindsight memory server.
// When --embedding-provider and --embedding-model are set, facts stored via
// hindsight_retain are automatically embedded via the OpenAI-compatible
// /v1/embeddings endpoint, and hindsight_recall can use dense=true for
// cosine-similarity semantic search over the dense vector space.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"

	"reasonix/internal/netclient"
)

// embeddingClient calls an OpenAI-compatible embeddings API.
type embeddingClient struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// newEmbeddingClient creates a client for the given provider configuration.
// baseURL should be the API root (e.g. "https://api.deepseek.com").
// apiKey is the raw key value. model is the embedding model name.
func newEmbeddingClient(baseURL, apiKey, model string) *embeddingClient {
	return &embeddingClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  netclient.DefaultClient(),
	}
}

// embedRequest is the JSON body for POST /v1/embeddings.
type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embedResponse is the JSON response from /v1/embeddings.
type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// Embed sends one or more texts to the embeddings API and returns the
// corresponding vectors. Each returned slice is the dense vector for the
// input at the same index.
func (ec *embeddingClient) Embed(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(embedRequest{Model: ec.model, Input: texts})
	if err != nil {
		return nil, fmt.Errorf("embed: marshal request: %w", err)
	}

	url := ec.baseURL + "/v1/embeddings"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embed: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if ec.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+ec.apiKey)
	}

	resp, err := ec.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("embed: API returned %d: %s", resp.StatusCode, string(b))
	}

	var er embedResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&er); err != nil {
		return nil, fmt.Errorf("embed: decode response: %w", err)
	}

	vectors := make([][]float64, len(er.Data))
	for _, d := range er.Data {
		if d.Index >= 0 && d.Index < len(vectors) {
			vectors[d.Index] = d.Embedding
		}
	}
	return vectors, nil
}

// embedOne is a convenience wrapper that embeds a single text and returns
// its dense vector, or nil on any error. Errors are logged.
func (ec *embeddingClient) embedOne(text string) []float64 {
	vecs, err := ec.Embed([]string{text})
	if err != nil {
		slog.Warn("hindsight: embed failed", "err", err)
		return nil
	}
	if len(vecs) > 0 {
		return vecs[0]
	}
	return nil
}

// denseCosine returns the cosine similarity between two dense float64 slices,
// which must have the same length. Returns 0 if either slice is nil or lengths
// differ.
func denseCosine(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	// math.Sqrt is exact; the previous hand-rolled Newton iteration seeded at
	// z=x left ~30% error for large norm-squared values (e.g. un-normalized
	// provider vectors), which mis-ranked and mis-thresholded dense search.
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// newEmbeddingClientFromEnv reads embedding configuration from environment
// variables. Returns nil when EMBEDDING_PROVIDER is not set.
//
// Environment variables:
//
//	EMBEDDING_PROVIDER   — base URL origin (e.g. https://api.deepseek.com)
//	EMBEDDING_MODEL      — model name (e.g. text-embedding-3-small)
//	EMBEDDING_API_KEY    — API key (falls back to DEEPSEEK_API_KEY, then OPENAI_API_KEY)
func newEmbeddingClientFromEnv() *embeddingClient {
	baseURL := os.Getenv("EMBEDDING_PROVIDER")
	if baseURL == "" {
		return nil
	}
	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return newEmbeddingClient(baseURL, apiKey, model)
}
