package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchPeers(t *testing.T) {
	want := []NetBirdPeer{
		{ID: "p1", Name: "peer-one", Hostname: "host1"},
		{ID: "p2", Name: "peer-two", Hostname: "host2"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/peers" {
			t.Errorf("got path %q, want /api/peers", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := fetchPeers(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d peers, want 2", len(got))
	}
	if got[0].ID != "p1" {
		t.Errorf("got first peer ID %q, want %q", got[0].ID, "p1")
	}
}

func TestGetPeer(t *testing.T) {
	want := NetBirdPeer{ID: "p1", Name: "peer-one", Hostname: "host1"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/peers/p1" {
			t.Errorf("got path %q, want /api/peers/p1", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := getPeer(cfg, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "p1" {
		t.Errorf("got ID %q, want %q", got.ID, "p1")
	}
}

func TestDeletePeer(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := deletePeer(cfg, "p1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("got method %q, want DELETE", gotMethod)
	}
	if gotPath != "/api/peers/p1" {
		t.Errorf("got path %q, want /api/peers/p1", gotPath)
	}
}

func TestUpdatePeer(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"name"`) {
			t.Errorf("body %q does not contain name field", string(body))
		}
		json.NewEncoder(w).Encode(NetBirdPeer{ID: "p1", Name: "renamed"})
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := updatePeer(cfg, "p1", "renamed", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("got method %q, want PUT", gotMethod)
	}
	if gotPath != "/api/peers/p1" {
		t.Errorf("got path %q, want /api/peers/p1", gotPath)
	}
	if got.Name != "renamed" {
		t.Errorf("got name %q, want %q", got.Name, "renamed")
	}
}

func TestListAccessiblePeers(t *testing.T) {
	want := []NetBirdPeer{
		{ID: "p2", Name: "peer-two"},
		{ID: "p3", Name: "peer-three"},
	}
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := listAccessiblePeers(cfg, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/peers/p1/accessible-peers" {
		t.Errorf("got path %q, want /api/peers/p1/accessible-peers", gotPath)
	}
	if len(got) != 2 {
		t.Fatalf("got %d peers, want 2", len(got))
	}
}

func TestFindPeerByName(t *testing.T) {
	peers := []NetBirdPeer{
		{ID: "p1", Name: "alpha", Hostname: "host-alpha"},
		{ID: "p2", Name: "beta", Hostname: "host-beta"},
	}

	// Match by Name
	p, err := findPeerByName(peers, "alpha")
	if err != nil {
		t.Fatalf("unexpected error finding by name: %v", err)
	}
	if p.ID != "p1" {
		t.Errorf("got ID %q, want %q", p.ID, "p1")
	}

	// Match by Hostname
	p, err = findPeerByName(peers, "host-beta")
	if err != nil {
		t.Fatalf("unexpected error finding by hostname: %v", err)
	}
	if p.ID != "p2" {
		t.Errorf("got ID %q, want %q", p.ID, "p2")
	}

	// Not found
	_, err = findPeerByName(peers, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent peer, got nil")
	}
	if !strings.Contains(err.Error(), "no peer found") {
		t.Errorf("unexpected error message: %v", err)
	}
}
