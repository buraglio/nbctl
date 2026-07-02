package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// NetBirdPeer represents a peer returned by GET /api/peers.
type NetBirdPeer struct {
	ID                          string         `json:"id"`
	Name                        string         `json:"name"`
	IP                          string         `json:"ip"`
	IPv6                        string         `json:"ipv6"`
	Connected                   bool           `json:"connected"`
	LastSeen                    string         `json:"last_seen"`
	OS                          string         `json:"os"`
	Version                     string         `json:"version"`
	Hostname                    string         `json:"hostname"`
	UserID                      string         `json:"user_id"`
	DNSLabel                    string         `json:"dns_label"`
	SSHEnabled                  bool           `json:"ssh_enabled"`
	LoginExpirationEnabled      bool           `json:"login_expiration_enabled"`
	LoginExpired                bool           `json:"login_expired"`
	LastLogin                   string         `json:"last_login"`
	InactivityExpirationEnabled bool           `json:"inactivity_expiration_enabled"`
	Groups                      []NetBirdGroup `json:"groups"`
}

// NetBirdGroup is the minimal group object embedded in peer responses.
type NetBirdGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ── API functions ─────────────────────────────────────────────────────────────

func nbPeerClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }

// fetchPeers retrieves all peers via GET /api/peers.
func fetchPeers(cfg *Config) ([]NetBirdPeer, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/peers", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbPeerClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var peers []NetBirdPeer
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("decode peers response: %w", err)
	}
	return peers, nil
}

// getPeer fetches a single peer by ID via GET /api/peers/{peerID}.
func getPeer(cfg *Config, peerID string) (NetBirdPeer, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/peers/"+peerID, nil)
	if err != nil {
		return NetBirdPeer{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbPeerClient().Do(req)
	if err != nil {
		return NetBirdPeer{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return NetBirdPeer{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var peer NetBirdPeer
	if err := json.NewDecoder(resp.Body).Decode(&peer); err != nil {
		return NetBirdPeer{}, fmt.Errorf("decode get-peer response: %w", err)
	}
	return peer, nil
}

// deletePeer removes a peer by ID via DELETE /api/peers/{peerID}.
func deletePeer(cfg *Config, peerID string) error {
	req, err := http.NewRequest(http.MethodDelete, cfg.NetBirdURL+"/api/peers/"+peerID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbPeerClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

// updatePeer updates a peer's settings via PUT /api/peers/{peerID}.
func updatePeer(cfg *Config, peerID, name string, sshEnabled, loginExpEnabled, inactivityExpEnabled bool) (NetBirdPeer, error) {
	payload := map[string]interface{}{
		"name":                         name,
		"ssh_enabled":                  sshEnabled,
		"login_expiration_enabled":     loginExpEnabled,
		"inactivity_expiration_enabled": inactivityExpEnabled,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return NetBirdPeer{}, err
	}

	req, err := http.NewRequest(http.MethodPut, cfg.NetBirdURL+"/api/peers/"+peerID, bytes.NewReader(encoded))
	if err != nil {
		return NetBirdPeer{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbPeerClient().Do(req)
	if err != nil {
		return NetBirdPeer{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return NetBirdPeer{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var peer NetBirdPeer
	if err := json.NewDecoder(resp.Body).Decode(&peer); err != nil {
		return NetBirdPeer{}, fmt.Errorf("decode update-peer response: %w", err)
	}
	return peer, nil
}

// listAccessiblePeers retrieves peers accessible from a given peer via GET /api/peers/{peerID}/accessible-peers.
func listAccessiblePeers(cfg *Config, peerID string) ([]NetBirdPeer, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/peers/"+peerID+"/accessible-peers", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbPeerClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var peers []NetBirdPeer
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("decode accessible-peers response: %w", err)
	}
	return peers, nil
}

// findPeerByName returns the first peer whose Name or Hostname matches.
func findPeerByName(peers []NetBirdPeer, name string) (NetBirdPeer, error) {
	for _, p := range peers {
		if p.Name == name || p.Hostname == name {
			return p, nil
		}
	}
	return NetBirdPeer{}, fmt.Errorf("no peer found with name %q", name)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func peerGroupNames(p NetBirdPeer) string {
	names := make([]string, 0, len(p.Groups))
	for _, g := range p.Groups {
		names = append(names, g.Name)
	}
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, ",")
}

func boolYN(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// ── peers command ─────────────────────────────────────────────────────────────

func runPeers(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl peers <list|show|update|delete|accessible> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		runPeersList(rest)
	case "show":
		runPeersShow(rest)
	case "update":
		runPeersUpdate(rest)
	case "delete":
		runPeersDelete(rest)
	case "accessible":
		runPeersAccessible(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown peers sub-command: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: nbctl peers <list|show|update|delete|accessible>")
		os.Exit(1)
	}
}

func runPeersList(args []string) {
	fs, cfg := newFlagSet("peers list")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	peers, err := fetchPeers(cfg)
	must(err, "list peers")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, peers)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "NAME\tHOSTNAME\tIP\tCONNECTED\tOS\tGROUPS")
	for _, p := range peers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			p.Name, dash(p.Hostname), dash(p.IP), boolYN(p.Connected), dash(p.OS), peerGroupNames(p))
	}
}

func runPeersShow(args []string) {
	fs, cfg := newFlagSet("peers show")
	peerName := fs.String("peer", "", "Peer name or hostname (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*peerName != "", "--peer is required")

	peers, err := fetchPeers(cfg)
	must(err, "fetch peers")

	p, err := findPeerByName(peers, *peerName)
	must(err, "find peer")

	peer, err := getPeer(cfg, p.ID)
	must(err, "get peer")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, peer)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "ID\t%s\n", peer.ID)
	fmt.Fprintf(w, "Name\t%s\n", peer.Name)
	fmt.Fprintf(w, "Hostname\t%s\n", dash(peer.Hostname))
	fmt.Fprintf(w, "IP\t%s\n", dash(peer.IP))
	fmt.Fprintf(w, "OS\t%s\n", dash(peer.OS))
	fmt.Fprintf(w, "Version\t%s\n", dash(peer.Version))
	fmt.Fprintf(w, "Connected\t%s\n", boolYN(peer.Connected))
	fmt.Fprintf(w, "LastSeen\t%s\n", dash(peer.LastSeen))
	fmt.Fprintf(w, "LastLogin\t%s\n", dash(peer.LastLogin))
	fmt.Fprintf(w, "SSH\t%s\n", boolYN(peer.SSHEnabled))
	fmt.Fprintf(w, "Groups\t%s\n", peerGroupNames(peer))
}

func runPeersUpdate(args []string) {
	fs, cfg := newFlagSet("peers update")
	peerName := fs.String("peer", "", "Peer name or hostname (required)")
	newName := fs.String("new-name", "", "New name for the peer (optional)")
	ssh := fs.Bool("ssh", false, "Enable SSH access")
	loginExp := fs.Bool("login-expiration", false, "Enable login expiration")
	inactivityExp := fs.Bool("inactivity-expiration", false, "Enable inactivity expiration")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*peerName != "", "--peer is required")

	peers, err := fetchPeers(cfg)
	must(err, "fetch peers")

	p, err := findPeerByName(peers, *peerName)
	must(err, "find peer")

	// Start with current values.
	name := p.Name
	if *newName != "" {
		name = *newName
	}
	sshEnabled := p.SSHEnabled
	loginExpEnabled := p.LoginExpirationEnabled
	inactivityExpEnabled := p.InactivityExpirationEnabled

	// Override only with explicitly provided flags.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "ssh":
			sshEnabled = *ssh
		case "login-expiration":
			loginExpEnabled = *loginExp
		case "inactivity-expiration":
			inactivityExpEnabled = *inactivityExp
		}
	})

	updated, err := updatePeer(cfg, p.ID, name, sshEnabled, loginExpEnabled, inactivityExpEnabled)
	must(err, "update peer")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, updated)
		return
	}
	logInfo("Updated peer %q (id %s)", updated.Name, updated.ID)
}

func runPeersDelete(args []string) {
	fs, cfg := newFlagSet("peers delete")
	peerName := fs.String("peer", "", "Peer name or hostname (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*peerName != "", "--peer is required")

	peers, err := fetchPeers(cfg)
	must(err, "fetch peers")

	p, err := findPeerByName(peers, *peerName)
	must(err, "find peer")

	must(deletePeer(cfg, p.ID), "delete peer")
	logInfo("Deleted peer %q (id %s)", *peerName, p.ID)
}

func runPeersAccessible(args []string) {
	fs, cfg := newFlagSet("peers accessible")
	peerName := fs.String("peer", "", "Peer name or hostname (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*peerName != "", "--peer is required")

	peers, err := fetchPeers(cfg)
	must(err, "fetch peers")

	p, err := findPeerByName(peers, *peerName)
	must(err, "find peer")

	accessible, err := listAccessiblePeers(cfg, p.ID)
	must(err, "list accessible peers")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, accessible)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "NAME\tHOSTNAME\tIP\tCONNECTED\tOS\tGROUPS")
	for _, ap := range accessible {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			ap.Name, dash(ap.Hostname), dash(ap.IP), boolYN(ap.Connected), dash(ap.OS), peerGroupNames(ap))
	}
}
