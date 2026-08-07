// Package marketplace: lobehub_client.go implements the LobeHub Marketplace
// API client — client registration, M2M OAuth2 token exchange, and skill
// listing/fetching from market.lobehub.com.
//
// Flow:
//  1. Register client: POST /api/v1/clients/register → client_id + client_secret
//  2. Exchange token: HS256 JWT → POST /oauth/token → access_token (3600s TTL)
//  3. Fetch skills: GET /api/v1/skills?page=N&pageSize=20&q=... with Bearer token
//
// All endpoints are at https://market.lobehub.com.
package marketplace

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"reasonix/internal/netclient"
)

var (
	lobeHubBaseURL = "https://market.lobehub.com"
)

const lobeHubUserAgent = "reasonix-hermes/1.7.0"

// LobeHubClient communicates with the LobeHub Marketplace API.
// It handles M2M OAuth2 auth and token caching transparently.
// Zero value is not ready; use NewLobeHubClient.
type LobeHubClient struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewLobeHubClient creates a client with the given credentials.
// If clientID or clientSecret is empty, the client will auto-register
// on the first API call via Register().
func NewLobeHubClient(clientID, clientSecret string) *LobeHubClient {
	return &LobeHubClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   netclient.DefaultClient(),
	}
}

// Register creates a new M2M client on the LobeHub marketplace.
// This requires no authentication. The returned credentials should be persisted
// for reuse (the same client_id continues to work for subsequent token exchanges).
func (c *LobeHubClient) Register(clientName, clientType, deviceID string) (clientID, clientSecret string, err error) {
	body := map[string]string{
		"clientName": clientName,
		"clientType": clientType,
		"deviceId":   deviceID,
	}
	b, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, lobeHubBaseURL+"/api/v1/clients/register", strings.NewReader(string(b)))
	if err != nil {
		return "", "", fmt.Errorf("lobehub register: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", lobeHubUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("lobehub register: %w", err)
	}
	defer resp.Body.Close()

	var reg struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Message      string `json:"message"`
		Success      *bool  `json:"success"`
		Error        *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&reg); err != nil {
		return "", "", fmt.Errorf("lobehub register: decode: %w", err)
	}
	if reg.Error != nil && reg.Error.Message != "" {
		return "", "", fmt.Errorf("lobehub register: %s", reg.Error.Message)
	}
	if reg.ClientID == "" || reg.ClientSecret == "" {
		return "", "", fmt.Errorf("lobehub register: server returned no credentials")
	}

	// Persist credentials in-memory for subsequent API calls.
	c.mu.Lock()
	c.clientID = reg.ClientID
	c.clientSecret = reg.ClientSecret
	c.accessToken = "" // clear any cached token
	c.mu.Unlock()

	return reg.ClientID, reg.ClientSecret, nil
}

// ClientID returns the current client ID (empty if not registered).
func (c *LobeHubClient) ClientID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clientID
}

// ClientSecret returns the current client secret (empty if not registered).
func (c *LobeHubClient) ClientSecret() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clientSecret
}

// authenticate obtains or reuses an OAuth2 access token via the M2M flow.
func (c *LobeHubClient) authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reuse cached token if it has >1 minute left.
	if c.accessToken != "" && time.Now().Add(time.Minute).Before(c.tokenExpiry) {
		return nil
	}

	if c.clientID == "" || c.clientSecret == "" {
		return fmt.Errorf("lobehub: not registered — call Register() first")
	}

	// Build HS256 JWT assertion.
	assertion, err := c.createJWT()
	if err != nil {
		return fmt.Errorf("lobehub auth: jwt: %w", err)
	}

	// Exchange JWT for access token.
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", assertion)

	req, err := http.NewRequest(http.MethodPost, lobeHubBaseURL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("lobehub auth: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", lobeHubUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("lobehub auth: %w", err)
	}
	defer resp.Body.Close()

	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&tok); err != nil {
		return fmt.Errorf("lobehub auth: decode: %w", err)
	}
	if tok.Error != "" {
		return fmt.Errorf("lobehub auth: %s — %s", tok.Error, tok.ErrorDesc)
	}
	if tok.AccessToken == "" {
		return fmt.Errorf("lobehub auth: server returned no access_token")
	}

	// Cache token with 60s safety margin.
	ttl := tok.ExpiresIn
	if ttl <= 0 {
		ttl = 3600
	}
	c.accessToken = tok.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(ttl-60) * time.Second)
	return nil
}

// createJWT builds an HS256 JWT assertion for the token exchange.
func (c *LobeHubClient) createJWT() (string, error) {
	now := time.Now()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload := fmt.Sprintf(
		`{"iss":%q,"sub":%q,"aud":%q,"jti":%q,"iat":%d,"exp":%d}`,
		c.clientID, c.clientID, lobeHubBaseURL+"/oauth/token",
		fmt.Sprintf("rxn-%d", now.UnixNano()),
		now.Unix(), now.Add(5*time.Minute).Unix(),
	)
	payloadEnc := base64.RawURLEncoding.EncodeToString([]byte(payload))

	signingInput := header + "." + payloadEnc

	mac := hmac.New(sha256.New, []byte(c.clientSecret))
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig, nil
}

// doGet makes an authenticated GET request to the LobeHub API.
func (c *LobeHubClient) doGet(path string, params url.Values) ([]byte, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	u := lobeHubBaseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("lobehub: %w", err)
	}

	c.mu.Lock()
	tok := c.accessToken
	c.mu.Unlock()

	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("User-Agent", lobeHubUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lobehub: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB max
	if err != nil {
		return nil, fmt.Errorf("lobehub: read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lobehub: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// AgentListResponse is the API response from GET /api/v1/agents.
type AgentListResponse struct {
	CurrentPage int                `json:"currentPage"`
	PageSize    int                `json:"pageSize"`
	TotalCount  int                `json:"totalCount"`
	TotalPages  int                `json:"totalPages"`
	Items       []LobeHubAgentItem `json:"items"`
}

// LobeHubAgentItem is one agent returned by the LobeHub marketplace API.
type LobeHubAgentItem struct {
	Identifier   string      `json:"identifier"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Author       AgentAuthor `json:"author"`
	Category     string      `json:"category"`
	Tags         []string    `json:"tags"`
	InstallCount int         `json:"installCount"`
	Version      string      `json:"versionName"`
	URL          string      `json:"url"`
	Avatar       string      `json:"avatar"`
	IsFeatured   bool        `json:"isFeatured"`
	IsOfficial   bool        `json:"isOfficial"`
	IsValidated  bool        `json:"isValidated"`
	Status       string      `json:"status"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
	ForkCount    int         `json:"forkCount"`
}

// AgentAuthor is the author metadata in an agent item.
type AgentAuthor struct {
	Name     string `json:"name"`
	UserName string `json:"userName"`
	Avatar   string `json:"avatar"`
}

// FetchAgents retrieves a single page of agents from the LobeHub marketplace.
func (c *LobeHubClient) FetchAgents(page, pageSize int, query, sort, order, category string) (*AgentListResponse, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))
	if query != "" {
		params.Set("q", query)
	}
	if sort != "" {
		params.Set("sort", sort)
	}
	if order != "" {
		params.Set("order", order)
	}
	if category != "" {
		params.Set("category", category)
	}

	body, err := c.doGet("/api/v1/agents", params)
	if err != nil {
		return nil, err
	}

	var resp AgentListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("lobehub: decode agents: %w", err)
	}
	return &resp, nil
}

// FetchAllAgents retrieves all agents from the LobeHub marketplace by
// paginating through all pages. Max pageSize is 20.
func (c *LobeHubClient) FetchAllAgents(pageSize int, query, sort, order, category string) ([]LobeHubAgentItem, error) {
	if pageSize <= 0 || pageSize > 20 {
		pageSize = 20
	}

	var all []LobeHubAgentItem
	page := 1
	for {
		resp, err := c.FetchAgents(page, pageSize, query, sort, order, category)
		if err != nil {
			return nil, fmt.Errorf("lobehub: page %d: %w", page, err)
		}
		all = append(all, resp.Items...)
		if page >= resp.TotalPages || len(resp.Items) == 0 {
			break
		}
		page++
		// Rate-limit: small delay between pages to be courteous.
		time.Sleep(200 * time.Millisecond)
	}
	return all, nil
}

// FetchSkills calls FetchAgents. Deprecated — kept for backward compat.
func (c *LobeHubClient) FetchSkills(page, pageSize int, query, sort, order, category string) (*AgentListResponse, error) {
	return c.FetchAgents(page, pageSize, query, sort, order, category)
}

// FetchAllSkills calls FetchAllAgents. Deprecated — kept for backward compat.
func (c *LobeHubClient) FetchAllSkills(pageSize int, query, sort, order, category string) ([]LobeHubAgentItem, error) {
	return c.FetchAllAgents(pageSize, query, sort, order, category)
}

// ToEntry converts a LobeHub marketplace agent to a Reasonix registry Entry.
func (s *LobeHubAgentItem) ToEntry() Entry {
	u := s.URL
	if u == "" {
		u = "https://lobehub.com/agents/" + s.Identifier
	}

	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}

	author := s.Author.Name
	if author == "" {
		author = s.Author.UserName
	}

	return Entry{
		Name:        s.Name,
		Description: s.Description,
		URL:         u,
		Author:      author,
		Tags:        tags,
		Rating:      0,
	}
}
