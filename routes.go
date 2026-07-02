package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

// NetBirdRoute represents a network route managed by NetBird.
type NetBirdRoute struct {
	ID                  string   `json:"id"`
	Description         string   `json:"description"`
	NetworkID           string   `json:"network_id"`
	Enabled             bool     `json:"enabled"`
	Peer                string   `json:"peer"`
	PeerGroups          []string `json:"peer_groups"`
	Network             string   `json:"network"`
	Domains             []string `json:"domains"`
	Metric              int      `json:"metric"`
	Masquerade          bool     `json:"masquerade"`
	Groups              []string `json:"groups"`
	KeepRoute           bool     `json:"keep_route"`
	AccessControlGroups []string `json:"access_control_groups"`
}

func nbRoutesClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }

// listRoutes returns all routes via GET /api/routes.
func listRoutes(cfg *Config) ([]NetBirdRoute, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/routes", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbRoutesClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result []NetBirdRoute
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode routes response: %w", err)
	}
	return result, nil
}

// getRoute fetches a single route by ID via GET /api/routes/{routeID}.
func getRoute(cfg *Config, routeID string) (NetBirdRoute, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/routes/"+routeID, nil)
	if err != nil {
		return NetBirdRoute{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbRoutesClient().Do(req)
	if err != nil {
		return NetBirdRoute{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdRoute{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result NetBirdRoute
	if err := json.Unmarshal(body, &result); err != nil {
		return NetBirdRoute{}, fmt.Errorf("decode route response: %w", err)
	}
	return result, nil
}

// deleteRoute removes a route via DELETE /api/routes/{routeID}.
func deleteRoute(cfg *Config, routeID string) error {
	req, err := http.NewRequest(http.MethodDelete, cfg.NetBirdURL+"/api/routes/"+routeID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbRoutesClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}

// createRoute creates a new route via POST /api/routes.
func createRoute(cfg *Config, r NetBirdRoute) (NetBirdRoute, error) {
	encoded, err := json.Marshal(r)
	if err != nil {
		return NetBirdRoute{}, err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.NetBirdURL+"/api/routes", bytes.NewReader(encoded))
	if err != nil {
		return NetBirdRoute{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbRoutesClient().Do(req)
	if err != nil {
		return NetBirdRoute{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdRoute{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result NetBirdRoute
	if err := json.Unmarshal(body, &result); err != nil {
		return NetBirdRoute{}, fmt.Errorf("decode create-route response: %w", err)
	}
	return result, nil
}

// updateRoute replaces a route via PUT /api/routes/{routeID}.
func updateRoute(cfg *Config, routeID string, r NetBirdRoute) (NetBirdRoute, error) {
	encoded, err := json.Marshal(r)
	if err != nil {
		return NetBirdRoute{}, err
	}

	req, err := http.NewRequest(http.MethodPut, cfg.NetBirdURL+"/api/routes/"+routeID, bytes.NewReader(encoded))
	if err != nil {
		return NetBirdRoute{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbRoutesClient().Do(req)
	if err != nil {
		return NetBirdRoute{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdRoute{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result NetBirdRoute
	if err := json.Unmarshal(body, &result); err != nil {
		return NetBirdRoute{}, fmt.Errorf("decode update-route response: %w", err)
	}
	return result, nil
}

// ── routes command ────────────────────────────────────────────────────────────

func runRoutes(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl routes <list|show|create|update|delete> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		runRoutesList(rest)
	case "show":
		runRoutesShow(rest)
	case "create":
		runRoutesCreate(rest)
	case "update":
		runRoutesUpdate(rest)
	case "delete":
		runRoutesDelete(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown routes sub-command: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: nbctl routes <list|show|create|update|delete>")
		os.Exit(1)
	}
}

func runRoutesList(args []string) {
	fs, cfg := newFlagSet("routes list")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	routes, err := listRoutes(cfg)
	must(err, "list routes")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, routes)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "ID\tNETWORK_ID\tNETWORK\tPEER\tMETRIC\tENABLED\tMASQUERADE")
	for _, r := range routes {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%v\t%v\n",
			r.ID, dash(r.NetworkID), dash(r.Network), dash(r.Peer), r.Metric, r.Enabled, r.Masquerade)
	}
}

func runRoutesShow(args []string) {
	fs, cfg := newFlagSet("routes show")
	routeID := fs.String("id", "", "Route ID (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*routeID != "", "--id is required")

	r, err := getRoute(cfg, *routeID)
	must(err, "get route")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, r)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "ID\t%s\n", r.ID)
	fmt.Fprintf(w, "NetworkID\t%s\n", dash(r.NetworkID))
	fmt.Fprintf(w, "Network\t%s\n", dash(r.Network))
	fmt.Fprintf(w, "Description\t%s\n", dash(r.Description))
	fmt.Fprintf(w, "Peer\t%s\n", dash(r.Peer))
	fmt.Fprintf(w, "Metric\t%d\n", r.Metric)
	fmt.Fprintf(w, "Enabled\t%v\n", r.Enabled)
	fmt.Fprintf(w, "Masquerade\t%v\n", r.Masquerade)
	fmt.Fprintf(w, "KeepRoute\t%v\n", r.KeepRoute)
}

func runRoutesCreate(args []string) {
	fs, cfg := newFlagSet("routes create")
	networkID := fs.String("network-id", "", "Network ID")
	network := fs.String("network", "", "Network CIDR (e.g. 10.0.0.0/24)")
	description := fs.String("description", "", "Route description")
	peer := fs.String("peer", "", "Peer ID that serves as gateway")
	metric := fs.Int("metric", 9999, "Route metric")
	masquerade := fs.Bool("masquerade", false, "Enable masquerading")
	enabled := fs.Bool("enabled", true, "Route is enabled")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	r := NetBirdRoute{
		NetworkID:   *networkID,
		Network:     *network,
		Description: *description,
		Peer:        *peer,
		Metric:      *metric,
		Masquerade:  *masquerade,
		Enabled:     *enabled,
		PeerGroups:  []string{},
		Groups:      []string{},
		Domains:     []string{},
	}

	created, err := createRoute(cfg, r)
	must(err, "create route")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, created)
		return
	}
	logInfo("Created route %s (network %s)", created.ID, created.Network)
}

func runRoutesUpdate(args []string) {
	fs, cfg := newFlagSet("routes update")
	routeID := fs.String("id", "", "Route ID to update (required)")
	networkID := fs.String("network-id", "", "Network ID")
	network := fs.String("network", "", "Network CIDR")
	description := fs.String("description", "", "Route description")
	peer := fs.String("peer", "", "Peer ID")
	metric := fs.Int("metric", 9999, "Route metric")
	masquerade := fs.Bool("masquerade", false, "Enable masquerading")
	enabled := fs.Bool("enabled", true, "Route is enabled")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*routeID != "", "--id is required")

	// Fetch the current route and merge only explicitly provided flags.
	current, err := getRoute(cfg, *routeID)
	must(err, "get route")

	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })

	if visited["network-id"] {
		current.NetworkID = *networkID
	}
	if visited["network"] {
		current.Network = *network
	}
	if visited["description"] {
		current.Description = *description
	}
	if visited["peer"] {
		current.Peer = *peer
	}
	if visited["metric"] {
		current.Metric = *metric
	}
	if visited["masquerade"] {
		current.Masquerade = *masquerade
	}
	if visited["enabled"] {
		current.Enabled = *enabled
	}

	updated, err := updateRoute(cfg, *routeID, current)
	must(err, "update route")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, updated)
		return
	}
	logInfo("Updated route %s", updated.ID)
}

func runRoutesDelete(args []string) {
	fs, cfg := newFlagSet("routes delete")
	routeID := fs.String("id", "", "Route ID to delete (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*routeID != "", "--id is required")

	must(deleteRoute(cfg, *routeID), "delete route")
	logInfo("Deleted route %s", *routeID)
}
