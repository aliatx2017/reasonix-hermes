package marketplace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterClient(t *testing.T) {
	// Cannot use t.Parallel() — mutates package-level lobeHubBaseURL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/clients/register" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"client_id":     "test-client-id",
			"client_secret": "test-client-secret",
			"message":       "ok",
		})
	}))
	defer srv.Close()

	// Override the base URL for testing.
	orig := lobeHubBaseURL
	defer func() { lobeHubBaseURL = orig }()
	lobeHubBaseURL = srv.URL

	client := NewLobeHubClient("", "")
	cid, csec, err := client.Register("test-app", "cli", "test-device")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if cid != "test-client-id" {
		t.Errorf("client_id = %q, want test-client-id", cid)
	}
	if csec != "test-client-secret" {
		t.Errorf("client_secret = %q, want test-client-secret", csec)
	}
}

func TestAuthAndFetchAgents(t *testing.T) {
	// Cannot use t.Parallel() — mutates package-level lobeHubBaseURL
	page := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/oauth/token":
			// Verify it's a proper form-encoded request.
			if err := r.ParseForm(); err != nil {
				t.Errorf("token parse: %v", err)
			}
			if r.Form.Get("grant_type") != "client_credentials" {
				t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
			}
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-access-token",
				"expires_in":   3600,
				"token_type":   "Bearer",
			})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/agents"):
			// Verify auth header.
			if got := r.Header.Get("Authorization"); got != "Bearer test-access-token" {
				t.Errorf("Authorization = %q", got)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			page++
			resp := AgentListResponse{
				CurrentPage: page,
				PageSize:    2,
				TotalCount:  3,
				TotalPages:  2,
			}
			if page == 1 {
				resp.Items = []LobeHubAgentItem{
					{Name: "agent-a", Identifier: "test.agent-a", Author: AgentAuthor{Name: "alice"}, InstallCount: 10, Tags: []string{"go"}},
					{Name: "agent-b", Identifier: "test.agent-b", Author: AgentAuthor{Name: "bob"}, InstallCount: 5, Tags: []string{"python"}},
				}
			} else {
				resp.Items = []LobeHubAgentItem{
					{Name: "agent-c", Identifier: "test.agent-c", Author: AgentAuthor{Name: "carol"}, InstallCount: 1, Tags: []string{}},
				}
			}
			json.NewEncoder(w).Encode(resp)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	orig := lobeHubBaseURL
	defer func() { lobeHubBaseURL = orig }()
	lobeHubBaseURL = srv.URL

	client := NewLobeHubClient("test-client-id", "test-client-secret")
	client.httpClient = srv.Client() // ensure same transport

	// Full paginated fetch.
	all, err := client.FetchAllAgents(2, "", "", "", "")
	if err != nil {
		t.Fatalf("FetchAllAgents: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 total, got %d", len(all))
	}
	if all[0].Name != "agent-a" {
		t.Errorf("first = %q", all[0].Name)
	}
	if all[2].Name != "agent-c" {
		t.Errorf("third = %q", all[2].Name)
	}

	// Verify FetchSkills (deprecated compat) also works.
	page = 0 // reset for compat call
	all2, err := client.FetchAllSkills(2, "", "", "", "")
	if err != nil {
		t.Fatalf("FetchAllSkills compat: %v", err)
	}
	if len(all2) != 3 {
		t.Fatalf("compat: expected 3 total, got %d", len(all2))
	}
}

func TestToEntry(t *testing.T) {
	t.Parallel()
	s := LobeHubAgentItem{
		Name:        "My Agent",
		Description: "A test agent",
		Author:      AgentAuthor{Name: "test-author", UserName: "testuser"},
		Tags:        []string{"go", "testing"},
		Identifier:  "test.my-agent",
		URL:         "https://example.com/agent",
	}
	e := s.ToEntry()
	if e.Name != "My Agent" {
		t.Errorf("Name = %q", e.Name)
	}
	if e.URL != "https://example.com/agent" {
		t.Errorf("URL = %q", e.URL)
	}
	if e.Author != "test-author" {
		t.Errorf("Author = %q", e.Author)
	}
	if e.Rating != 0 {
		t.Errorf("Rating = %f, want 0 (no rating in agents API)", e.Rating)
	}
	if len(e.Tags) != 2 {
		t.Errorf("Tags = %v", e.Tags)
	}

	// No URL falls back to lobehub.com/agents/<identifier>.
	s2 := LobeHubAgentItem{
		Name:       "No URL Agent",
		Identifier: "test.no-url",
		Author:     AgentAuthor{Name: "anon"},
	}
	e2 := s2.ToEntry()
	if e2.URL != "https://lobehub.com/agents/test.no-url" {
		t.Errorf("no-URL fallback = %q", e2.URL)
	}

	// Author falls back to UserName when Name is empty.
	s3 := LobeHubAgentItem{
		Name:   "UserFallback",
		Author: AgentAuthor{UserName: "handle42"},
	}
	e3 := s3.ToEntry()
	if e3.Author != "handle42" {
		t.Errorf("author fallback = %q", e3.Author)
	}
}

func TestMergeFromLobeHub(t *testing.T) {
	t.Parallel()
	reg := &Registry{
		entries: []Entry{
			{Name: "existing-agent", Description: "already here", Rating: 4.0},
		},
	}

	agents := []LobeHubAgentItem{
		{Name: "existing-agent", Description: "duplicate"},                                // should be skipped
		{Name: "new-agent", Description: "fresh from lobehub"},                            // should be added
		{Name: "", Description: "no name"},                                                // should be skipped
		{Name: "EXISTING-AGENT", Description: "case-insensitive duplicate"},              // should be skipped
	}

	added := reg.MergeFromLobeHub(agents)
	if added != 1 {
		t.Errorf("added = %d, want 1", added)
	}
	if reg.Len() != 2 {
		t.Errorf("Len = %d, want 2", reg.Len())
	}

	got := reg.ByName("new-agent")
	if got == nil {
		t.Fatal("new-agent not found")
	}
	if got.Rating != 0 {
		t.Errorf("Rating = %f, want 0", got.Rating)
	}
}
