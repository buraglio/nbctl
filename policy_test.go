package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListPolicies(t *testing.T) {
	want := []NetBirdPolicy{
		{ID: "p1", Name: "allow-all", Enabled: true, Rules: []NetBirdPolicyRule{{Name: "r1"}}},
		{ID: "p2", Name: "block-ssh", Enabled: false, Rules: []NetBirdPolicyRule{}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/policies" {
			t.Errorf("got path %q, want /api/policies", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := listPolicies(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d policies, want 2", len(got))
	}
}

func TestGetPolicy(t *testing.T) {
	want := NetBirdPolicy{ID: "p1", Name: "allow-all", Enabled: true}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/policies/p1" {
			t.Errorf("got path %q, want /api/policies/p1", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := getPolicy(cfg, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "allow-all" {
		t.Errorf("got name %q, want %q", got.Name, "allow-all")
	}
}

func TestDeletePolicy(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := deletePolicy(cfg, "p1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("got method %q, want DELETE", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/p1") {
		t.Errorf("got path %q, want suffix /p1", gotPath)
	}
}

func TestCreatePolicy(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		result := NetBirdPolicy{ID: "p3", Name: "new-policy", Enabled: true}
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	input := NetBirdPolicy{
		Name:    "new-policy",
		Enabled: true,
		Rules: []NetBirdPolicyRule{
			{Name: "rule1", Action: "accept", Protocol: "all", Bidirectional: true,
				Sources: []string{"grp1"}, Destinations: []string{"grp2"}},
		},
	}
	got, err := createPolicy(cfg, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("got method %q, want POST", gotMethod)
	}
	if gotPath != "/api/policies" {
		t.Errorf("got path %q, want /api/policies", gotPath)
	}
	if _, ok := gotBody["name"]; !ok {
		t.Error("request body missing 'name' field")
	}
	if got.ID != "p3" {
		t.Errorf("got ID %q, want %q", got.ID, "p3")
	}
}

func TestUpdatePolicy(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		result := NetBirdPolicy{ID: "p1", Name: "allow-all", Enabled: false}
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	input := NetBirdPolicy{Name: "allow-all", Enabled: false, Rules: []NetBirdPolicyRule{}}
	got, err := updatePolicy(cfg, "p1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("got method %q, want PUT", gotMethod)
	}
	if gotPath != "/api/policies/p1" {
		t.Errorf("got path %q, want /api/policies/p1", gotPath)
	}
	if got.Enabled {
		t.Error("expected Enabled=false, got true")
	}
}
