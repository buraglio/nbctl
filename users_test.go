package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListUsers(t *testing.T) {
	want := []NetBirdUser{
		{ID: "u1", Email: "alice@example.com", Name: "Alice"},
		{ID: "u2", Email: "bob@example.com", Name: "Bob"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/users" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := listUsers(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d users, want 2", len(got))
	}
	if got[0].Email != "alice@example.com" {
		t.Errorf("got email %q, want %q", got[0].Email, "alice@example.com")
	}
}

func TestCreateUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/users" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["email"]; !ok {
			t.Errorf("request body missing 'email' field")
		}
		u := NetBirdUser{ID: "u3", Email: body["email"].(string), Name: "Charlie"}
		json.NewEncoder(w).Encode(u)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := createUser(cfg, "charlie@example.com", "Charlie", "user", false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "charlie@example.com" {
		t.Errorf("got email %q, want %q", got.Email, "charlie@example.com")
	}
}

func TestDeleteUser(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := deleteUser(cfg, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("got method %q, want DELETE", gotMethod)
	}
	if gotPath != "/api/users/u1" {
		t.Errorf("got path %q, want /api/users/u1", gotPath)
	}
}

func TestInviteUser(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := inviteUser(cfg, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("got method %q, want POST", gotMethod)
	}
	if gotPath != "/api/users/u1/invite" {
		t.Errorf("got path %q, want /api/users/u1/invite", gotPath)
	}
}

func TestApproveUser(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := approveUser(cfg, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("got method %q, want POST", gotMethod)
	}
	if gotPath != "/api/users/u1/approve" {
		t.Errorf("got path %q, want /api/users/u1/approve", gotPath)
	}
}

func TestFindUserByEmail(t *testing.T) {
	users := []NetBirdUser{
		{ID: "u1", Email: "alice@example.com", Name: "Alice"},
		{ID: "u2", Email: "bob@example.com", Name: "Bob"},
	}

	// match
	u, err := findUserByEmail(users, "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "u1" {
		t.Errorf("got ID %q, want %q", u.ID, "u1")
	}

	// not found
	_, err = findUserByEmail(users, "nobody@example.com")
	if err == nil {
		t.Error("expected error for missing user, got nil")
	}
}
