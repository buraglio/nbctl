package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListPostureChecks(t *testing.T) {
	want := []NetBirdPostureCheck{
		{ID: "pc1", Name: "Linux version check"},
		{ID: "pc2", Name: "Geo block"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/posture-checks" {
			t.Errorf("got path %q, want /api/posture-checks", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := listPostureChecks(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d checks, want 2", len(got))
	}
}

func TestGetPostureCheck(t *testing.T) {
	want := NetBirdPostureCheck{ID: "pc1", Name: "Linux version check"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/posture-checks/pc1" {
			t.Errorf("got path %q, want /api/posture-checks/pc1", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := getPostureCheck(cfg, "pc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "pc1" {
		t.Errorf("got ID %q, want %q", got.ID, "pc1")
	}
}

func TestDeletePostureCheck(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := deletePostureCheck(cfg, "pc1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("got method %q, want DELETE", gotMethod)
	}
	if gotPath != "/api/posture-checks/pc1" {
		t.Errorf("got path %q, want /api/posture-checks/pc1", gotPath)
	}
}
