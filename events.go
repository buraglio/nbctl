package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

// NetBirdAuditEvent represents a single audit log entry from the NetBird API.
type NetBirdAuditEvent struct {
	ID        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Activity  string                 `json:"activity"`
	TargetID  string                 `json:"target_id,omitempty"`
	Target    map[string]interface{} `json:"target,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
	Initiator struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"initiator"`
}

func nbEventsClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }

// listAuditEvents returns all audit events via GET /api/events/audit.
func listAuditEvents(cfg *Config) ([]NetBirdAuditEvent, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/events/audit", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbEventsClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, b)
	}
	var result []NetBirdAuditEvent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode audit events response: %w", err)
	}
	return result, nil
}

// ── events command ─────────────────────────────────────────────────────────────

func runEvents(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl events <audit|traffic> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "audit":
		runEventsAudit(rest)
	case "traffic":
		runEventsTraffic(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown events sub-command: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: nbctl events <audit|traffic>")
		os.Exit(1)
	}
}

func runEventsAudit(args []string) {
	fs, cfg := newFlagSet("events audit")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	events, err := listAuditEvents(cfg)
	must(err, "list audit events")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, events)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "TIMESTAMP\tACTIVITY\tINITIATOR\tTARGET_ID")
	for _, e := range events {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			dash(e.Timestamp), dash(e.Activity), dash(e.Initiator.Email), dash(e.TargetID))
	}
}

func runEventsTraffic(args []string) {
	logInfo("Traffic events are a cloud-only experimental feature — use the NetBird dashboard or API directly")
	os.Exit(0)
}
