package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListGroups(t *testing.T) {
	want := []NetBirdGroupFull{
		{ID: "g1", Name: "mygroup", PeersCount: 2},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/groups" {
			t.Errorf("got path %q, want /api/groups", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := listGroups(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "mygroup" {
		t.Errorf("unexpected groups: %v", got)
	}
}

func TestGetGroup(t *testing.T) {
	want := NetBirdGroupFull{ID: "g1", Name: "mygroup"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/groups/g1" {
			t.Errorf("got path %q, want /api/groups/g1", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := getGroup(cfg, "g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "g1" {
		t.Errorf("got ID %q, want %q", got.ID, "g1")
	}
}

func TestCreateGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("got method %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/groups" {
			t.Errorf("got path %q, want /api/groups", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"name"`) {
			t.Errorf("body %q does not contain \"name\"", string(b))
		}
		json.NewEncoder(w).Encode(NetBirdGroupFull{ID: "g2", Name: "newgroup"})
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := createGroup(cfg, "newgroup", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "newgroup" {
		t.Errorf("got name %q, want %q", got.Name, "newgroup")
	}
}

func TestUpdateGroup(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(NetBirdGroupFull{ID: "g1", Name: "updated"})
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	_, err := updateGroup(cfg, "g1", "updated", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("got method %q, want PUT", gotMethod)
	}
	if gotPath != "/api/groups/g1" {
		t.Errorf("got path %q, want /api/groups/g1", gotPath)
	}
}

func TestDeleteGroup(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	if err := deleteGroup(cfg, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("got method %q, want DELETE", gotMethod)
	}
	if gotPath != "/api/groups/g1" {
		t.Errorf("got path %q, want /api/groups/g1", gotPath)
	}
}

func TestFindGroupByName(t *testing.T) {
	groups := []NetBirdGroupFull{
		{ID: "g1", Name: "alpha"},
		{ID: "g2", Name: "beta"},
	}

	got, err := findGroupByName(groups, "alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "g1" {
		t.Errorf("got ID %q, want %q", got.ID, "g1")
	}

	_, err = findGroupByName(groups, "notfound")
	if err == nil {
		t.Error("expected error for not-found group, got nil")
	}
}
