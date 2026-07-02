package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRoutes(t *testing.T) {
	want := []NetBirdRoute{
		{ID: "r1", NetworkID: "net1", Network: "10.0.0.0/24", Enabled: true},
		{ID: "r2", NetworkID: "net2", Network: "192.168.1.0/24", Enabled: false},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/routes" {
			t.Errorf("got path %q, want /api/routes", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := listRoutes(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d routes, want 2", len(got))
	}
}

func TestGetRoute(t *testing.T) {
	want := NetBirdRoute{ID: "r1", NetworkID: "net1", Network: "10.0.0.0/24"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/routes/r1" {
			t.Errorf("got path %q, want /api/routes/r1", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := getRoute(cfg, "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "r1" {
		t.Errorf("got ID %q, want %q", got.ID, "r1")
	}
}

func TestDeleteRoute(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := deleteRoute(cfg, "r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("got method %q, want DELETE", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/r1") {
		t.Errorf("got path %q, want suffix /r1", gotPath)
	}
}

func TestCreateRoute(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		result := NetBirdRoute{ID: "r3", Network: "172.16.0.0/12", Enabled: true}
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	input := NetBirdRoute{Network: "172.16.0.0/12", Metric: 100, Enabled: true}
	got, err := createRoute(cfg, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("got method %q, want POST", gotMethod)
	}
	if gotPath != "/api/routes" {
		t.Errorf("got path %q, want /api/routes", gotPath)
	}
	if _, ok := gotBody["network"]; !ok {
		t.Error("request body missing 'network' field")
	}
	if got.ID != "r3" {
		t.Errorf("got ID %q, want %q", got.ID, "r3")
	}
}

func TestUpdateRoute(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		result := NetBirdRoute{ID: "r1", Network: "10.0.0.0/24", Metric: 50}
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	input := NetBirdRoute{Network: "10.0.0.0/24", Metric: 50, Enabled: true}
	got, err := updateRoute(cfg, "r1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("got method %q, want PUT", gotMethod)
	}
	if gotPath != "/api/routes/r1" {
		t.Errorf("got path %q, want /api/routes/r1", gotPath)
	}
	if got.Metric != 50 {
		t.Errorf("got metric %d, want 50", got.Metric)
	}
}
