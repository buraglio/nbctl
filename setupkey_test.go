package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListSetupKeys(t *testing.T) {
	want := []NetBirdSetupKey{
		{ID: "k1", Name: "ci-key", Type: "reusable", State: "valid"},
		{ID: "k2", Name: "dev-key", Type: "one-off", State: "valid"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/setup-keys" {
			t.Errorf("got path %q, want /api/setup-keys", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := listSetupKeys(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d keys, want 2", len(got))
	}
	if got[0].Name != "ci-key" {
		t.Errorf("got name %q, want %q", got[0].Name, "ci-key")
	}
}

func TestGetSetupKey(t *testing.T) {
	want := NetBirdSetupKey{ID: "k1", Name: "ci-key", Type: "reusable"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/setup-keys/k1" {
			t.Errorf("got path %q, want /api/setup-keys/k1", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := getSetupKey(cfg, "k1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "k1" {
		t.Errorf("got ID %q, want %q", got.ID, "k1")
	}
}

func TestCreateSetupKey(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		result := NetBirdSetupKey{ID: "k3", Name: "new-key", Type: "one-off", Key: "nbsk_secret"}
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := createSetupKey(cfg, "new-key", "one-off", 86400, false, false, []string{}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("got method %q, want POST", gotMethod)
	}
	if gotPath != "/api/setup-keys" {
		t.Errorf("got path %q, want /api/setup-keys", gotPath)
	}
	if _, ok := gotBody["name"]; !ok {
		t.Error("request body missing 'name' field")
	}
	if _, ok := gotBody["type"]; !ok {
		t.Error("request body missing 'type' field")
	}
	if got.Key != "nbsk_secret" {
		t.Errorf("got key %q, want %q", got.Key, "nbsk_secret")
	}
}

func TestRevokeSetupKey(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(NetBirdSetupKey{ID: "k1", Revoked: true})
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := revokeSetupKey(cfg, "k1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("got method %q, want PUT", gotMethod)
	}
	if gotPath != "/api/setup-keys/k1" {
		t.Errorf("got path %q, want /api/setup-keys/k1", gotPath)
	}
	revoked, _ := gotBody["revoked"].(bool)
	if !revoked {
		t.Errorf("body should have revoked:true, got %v", gotBody)
	}
}

func TestDeleteSetupKey(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := deleteSetupKey(cfg, "k1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("got method %q, want DELETE", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/k1") {
		t.Errorf("got path %q, want suffix /k1", gotPath)
	}
}
