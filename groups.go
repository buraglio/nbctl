package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// stringSliceFlag is a flag.Value that accumulates multiple --flag values.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	if s == nil || len(*s) == 0 {
		return ""
	}
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// NetBirdGroupPeer is a peer summary embedded inside a group response.
type NetBirdGroupPeer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Connected bool   `json:"connected"`
}

// NetBirdGroupFull is the full group object returned by the NetBird API.
type NetBirdGroupFull struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	PeersCount     int                `json:"peers_count"`
	ResourcesCount int                `json:"resources_count"`
	Issued         string             `json:"issued"`
	Peers          []NetBirdGroupPeer `json:"peers"`
}

func nbGroupsClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }

// listGroups returns all groups via GET /api/groups.
func listGroups(cfg *Config) ([]NetBirdGroupFull, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/groups", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbGroupsClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, b)
	}
	var result []NetBirdGroupFull
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode groups response: %w", err)
	}
	return result, nil
}

// getGroup fetches a single group by ID via GET /api/groups/{groupID}.
func getGroup(cfg *Config, groupID string) (NetBirdGroupFull, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/groups/"+groupID, nil)
	if err != nil {
		return NetBirdGroupFull{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbGroupsClient().Do(req)
	if err != nil {
		return NetBirdGroupFull{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return NetBirdGroupFull{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, b)
	}
	var result NetBirdGroupFull
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return NetBirdGroupFull{}, fmt.Errorf("decode group response: %w", err)
	}
	return result, nil
}

// createGroup creates a new group via POST /api/groups.
func createGroup(cfg *Config, name string, peerIDs []string) (NetBirdGroupFull, error) {
	if peerIDs == nil {
		peerIDs = []string{}
	}
	payload := map[string]interface{}{"name": name, "peers": peerIDs}
	body, err := json.Marshal(payload)
	if err != nil {
		return NetBirdGroupFull{}, err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.NetBirdURL+"/api/groups", bytes.NewReader(body))
	if err != nil {
		return NetBirdGroupFull{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbGroupsClient().Do(req)
	if err != nil {
		return NetBirdGroupFull{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return NetBirdGroupFull{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, b)
	}
	var result NetBirdGroupFull
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return NetBirdGroupFull{}, fmt.Errorf("decode create-group response: %w", err)
	}
	return result, nil
}

// updateGroup replaces a group's name and peer list via PUT /api/groups/{groupID}.
func updateGroup(cfg *Config, groupID, name string, peerIDs []string) (NetBirdGroupFull, error) {
	if peerIDs == nil {
		peerIDs = []string{}
	}
	payload := map[string]interface{}{"name": name, "peers": peerIDs}
	body, err := json.Marshal(payload)
	if err != nil {
		return NetBirdGroupFull{}, err
	}

	req, err := http.NewRequest(http.MethodPut, cfg.NetBirdURL+"/api/groups/"+groupID, bytes.NewReader(body))
	if err != nil {
		return NetBirdGroupFull{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbGroupsClient().Do(req)
	if err != nil {
		return NetBirdGroupFull{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return NetBirdGroupFull{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, b)
	}
	var result NetBirdGroupFull
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return NetBirdGroupFull{}, fmt.Errorf("decode update-group response: %w", err)
	}
	return result, nil
}

// deleteGroup removes a group via DELETE /api/groups/{groupID}.
func deleteGroup(cfg *Config, groupID string) error {
	req, err := http.NewRequest(http.MethodDelete, cfg.NetBirdURL+"/api/groups/"+groupID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbGroupsClient().Do(req)
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

// findGroupByName does a linear scan of groups and returns the first match by name.
func findGroupByName(groups []NetBirdGroupFull, name string) (NetBirdGroupFull, error) {
	for _, g := range groups {
		if g.Name == name {
			return g, nil
		}
	}
	return NetBirdGroupFull{}, fmt.Errorf("group %q not found", name)
}

// ── groups command ─────────────────────────────────────────────────────────────

func runGroups(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl groups <list|show|create|update|delete> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		runGroupsList(rest)
	case "show":
		runGroupsShow(rest)
	case "create":
		runGroupsCreate(rest)
	case "update":
		runGroupsUpdate(rest)
	case "delete":
		runGroupsDelete(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown groups sub-command: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: nbctl groups <list|show|create|update|delete>")
		os.Exit(1)
	}
}

func runGroupsList(args []string) {
	fs, cfg := newFlagSet("groups list")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	groups, err := listGroups(cfg)
	must(err, "list groups")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, groups)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "ID\tNAME\tPEERS\tRESOURCES\tISSUED")
	for _, g := range groups {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
			g.ID, g.Name, g.PeersCount, g.ResourcesCount, dash(g.Issued))
	}
}

func runGroupsShow(args []string) {
	fs, cfg := newFlagSet("groups show")
	groupName := fs.String("group", "", "Group name to show")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*groupName != "", "--group is required")

	groups, err := listGroups(cfg)
	must(err, "list groups")

	found, err := findGroupByName(groups, *groupName)
	must(err, "find group")

	g, err := getGroup(cfg, found.ID)
	must(err, "get group")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, g)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\t%s\n", g.ID)
	fmt.Fprintf(w, "Name\t%s\n", g.Name)
	fmt.Fprintf(w, "Peers\t%d\n", g.PeersCount)
	fmt.Fprintf(w, "Resources\t%d\n", g.ResourcesCount)
	fmt.Fprintf(w, "Issued\t%s\n", dash(g.Issued))
	w.Flush()

	if len(g.Peers) > 0 {
		fmt.Println()
		pw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(pw, "PEER_ID\tPEER_NAME\tIP\tCONNECTED")
		for _, p := range g.Peers {
			fmt.Fprintf(pw, "%s\t%s\t%s\t%v\n", p.ID, p.Name, dash(p.IP), p.Connected)
		}
		pw.Flush()
	}
}

func runGroupsCreate(args []string) {
	fs, cfg := newFlagSet("groups create")
	name := fs.String("name", "", "Group name")
	var peers stringSliceFlag
	fs.Var(&peers, "peer", "Peer ID to add (repeatable)")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*name != "", "--name is required")

	g, err := createGroup(cfg, *name, []string(peers))
	must(err, "create group")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, g)
		return
	}
	logInfo("Created group %q (id %s)", g.Name, g.ID)
}

func runGroupsUpdate(args []string) {
	fs, cfg := newFlagSet("groups update")
	id := fs.String("id", "", "Group ID to update")
	name := fs.String("name", "", "New group name")
	var peers stringSliceFlag
	fs.Var(&peers, "peer", "Peer ID (repeatable; replaces the group's peer list)")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*id != "", "--id is required")
	require(*name != "", "--name is required")

	g, err := updateGroup(cfg, *id, *name, []string(peers))
	must(err, "update group")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, g)
		return
	}
	logInfo("Updated group %q (id %s)", g.Name, g.ID)
}

func runGroupsDelete(args []string) {
	fs, cfg := newFlagSet("groups delete")
	id := fs.String("id", "", "Group ID to delete")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*id != "", "--id is required")

	must(deleteGroup(cfg, *id), "delete group")
	logInfo("Deleted group %s", *id)
}
