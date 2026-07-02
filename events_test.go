package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAuditEvents(t *testing.T) {
	want := []NetBirdAuditEvent{
		{ID: "e1", Activity: "peer.added", Timestamp: "2026-01-01T00:00:00Z"},
		{ID: "e2", Activity: "user.login", Timestamp: "2026-01-02T00:00:00Z"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("got method %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/events/audit" {
			t.Errorf("got path %q, want /api/events/audit", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	cfg := &Config{NetBirdURL: srv.URL, NetBirdToken: "test"}
	got, err := listAuditEvents(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d events, want 2", len(got))
	}
}
