package marketplace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterClient(t *testing.T) {
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

func TestAuthAndFetchSkills(t *testing.T) {
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

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/skills"):
			// Verify auth header.
			if got := r.Header.Get("Authorization"); got != "Bearer test-access-token" {
				t.Errorf("Authorization = %q", got)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			page++
			resp := SkillListResponse{
				CurrentPage: page,
				PageSize:    2,
				TotalCount:  3,
				TotalPages:  2,
				Categories:  []string{"testing"},
			}
			if page == 1 {
				resp.Items = []LobeHubSkillItem{
					{Name: "skill-a", Identifier: "test.skill-a", Author: "alice", InstallCount: 10, RatingAvg: 4.5, Tags: []string{"go"}},
					{Name: "skill-b", Identifier: "test.skill-b", Author: "bob", InstallCount: 5, RatingAvg: 4.0, Tags: []string{"python"}},
				}
			} else {
				resp.Items = []LobeHubSkillItem{
					{Name: "skill-c", Identifier: "test.skill-c", Author: "carol", InstallCount: 1, RatingAvg: 0, Tags: []string{}},
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
	all, err := client.FetchAllSkills(2, "", "", "", "")
	if err != nil {
		t.Fatalf("FetchAllSkills: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 total, got %d", len(all))
	}
	if all[0].Name != "skill-a" {
		t.Errorf("first = %q", all[0].Name)
	}
	if all[2].Name != "skill-c" {
		t.Errorf("third = %q", all[2].Name)
	}
}

func TestToEntry(t *testing.T) {
	s := LobeHubSkillItem{
		Name:        "My Skill",
		Description: "A test skill",
		Author:      "test-author",
		Tags:        []string{"go", "testing"},
		RatingAvg:   4.5,
		Identifier:  "test.my-skill",
		Homepage:    "https://example.com/skill",
	}
	e := s.ToEntry()
	if e.Name != "My Skill" {
		t.Errorf("Name = %q", e.Name)
	}
	if e.URL != "https://example.com/skill" {
		t.Errorf("URL = %q", e.URL)
	}
	if e.Rating != 4.5 {
		t.Errorf("Rating = %f", e.Rating)
	}
	if len(e.Tags) != 2 {
		t.Errorf("Tags = %v", e.Tags)
	}

	// Validated skill with no ratings gets baseline.
	s2 := LobeHubSkillItem{
		Name:        "New Skill",
		Description: "Fresh",
		IsValidated: true,
		RatingAvg:   0,
	}
	e2 := s2.ToEntry()
	if e2.Rating != 4.0 {
		t.Errorf("validated skill with no ratings: Rating = %f, want 4.0", e2.Rating)
	}
}

func TestMergeFromLobeHub(t *testing.T) {
	reg := &Registry{
		entries: []Entry{
			{Name: "existing-skill", Description: "already here", Rating: 4.0},
		},
	}

	skills := []LobeHubSkillItem{
		{Name: "existing-skill", Description: "duplicate", RatingAvg: 5.0},          // should be skipped
		{Name: "new-skill", Description: "fresh from lobehub", RatingAvg: 4.8},    // should be added
		{Name: "", Description: "no name"},                                         // should be skipped
		{Name: "EXISTING-SKILL", Description: "case-insensitive duplicate", RatingAvg: 5.0}, // should be skipped
	}

	added := reg.MergeFromLobeHub(skills)
	if added != 1 {
		t.Errorf("added = %d, want 1", added)
	}
	if reg.Len() != 2 {
		t.Errorf("Len = %d, want 2", reg.Len())
	}

	got := reg.ByName("new-skill")
	if got == nil {
		t.Fatal("new-skill not found")
	}
	if got.Rating != 4.8 {
		t.Errorf("Rating = %f", got.Rating)
	}
}
