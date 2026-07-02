package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

// NetBirdSetupKey represents a setup key returned by the NetBird API.
type NetBirdSetupKey struct {
	ID         string   `json:"id"`
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`       // "one-off" or "reusable"
	ExpiresAt  string   `json:"expires_at"`
	Revoked    bool     `json:"revoked"`
	UsedTimes  int      `json:"used_times"`
	UsageLimit int      `json:"usage_limit"`
	LastUsed   string   `json:"last_used"`
	State      string   `json:"state"`
	AutoGroups []string `json:"auto_groups"`
	Ephemeral  bool     `json:"ephemeral"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

func nbSetupKeyClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }

// listSetupKeys returns all setup keys via GET /api/setup-keys.
func listSetupKeys(cfg *Config) ([]NetBirdSetupKey, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/setup-keys", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbSetupKeyClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result []NetBirdSetupKey
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode setup-keys response: %w", err)
	}
	return result, nil
}

// getSetupKey fetches a single setup key by ID via GET /api/setup-keys/{keyID}.
func getSetupKey(cfg *Config, keyID string) (NetBirdSetupKey, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/setup-keys/"+keyID, nil)
	if err != nil {
		return NetBirdSetupKey{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbSetupKeyClient().Do(req)
	if err != nil {
		return NetBirdSetupKey{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdSetupKey{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result NetBirdSetupKey
	if err := json.Unmarshal(body, &result); err != nil {
		return NetBirdSetupKey{}, fmt.Errorf("decode setup-key response: %w", err)
	}
	return result, nil
}

// createSetupKey creates a new setup key via POST /api/setup-keys.
// expiresIn is the number of seconds until expiry (86400–31536000).
func createSetupKey(cfg *Config, name, keyType string, expiresIn int, reusable, ephemeral bool, autoGroups []string, usageLimit int) (NetBirdSetupKey, error) {
	if keyType == "" {
		if reusable {
			keyType = "reusable"
		} else {
			keyType = "one-off"
		}
	}
	if autoGroups == nil {
		autoGroups = []string{}
	}
	payload := map[string]interface{}{
		"name":        name,
		"type":        keyType,
		"expires_in":  expiresIn,
		"auto_groups": autoGroups,
		"usage_limit": usageLimit,
		"ephemeral":   ephemeral,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return NetBirdSetupKey{}, err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.NetBirdURL+"/api/setup-keys", bytes.NewReader(encoded))
	if err != nil {
		return NetBirdSetupKey{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbSetupKeyClient().Do(req)
	if err != nil {
		return NetBirdSetupKey{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdSetupKey{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result NetBirdSetupKey
	if err := json.Unmarshal(body, &result); err != nil {
		return NetBirdSetupKey{}, fmt.Errorf("decode create-setup-key response: %w", err)
	}
	return result, nil
}

// revokeSetupKey marks a key as revoked via PUT /api/setup-keys/{keyID}.
func revokeSetupKey(cfg *Config, keyID string) error {
	payload := map[string]interface{}{
		"revoked":     true,
		"auto_groups": []string{},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, cfg.NetBirdURL+"/api/setup-keys/"+keyID, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbSetupKeyClient().Do(req)
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

// deleteSetupKey removes a setup key via DELETE /api/setup-keys/{keyID}.
func deleteSetupKey(cfg *Config, keyID string) error {
	req, err := http.NewRequest(http.MethodDelete, cfg.NetBirdURL+"/api/setup-keys/"+keyID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbSetupKeyClient().Do(req)
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

// ── setup-key command ─────────────────────────────────────────────────────────

func runSetupKey(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl setup-key <list|show|create|revoke|delete> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		runSetupKeyList(rest)
	case "show":
		runSetupKeyShow(rest)
	case "create":
		runSetupKeyCreate(rest)
	case "revoke":
		runSetupKeyRevoke(rest)
	case "delete":
		runSetupKeyDelete(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown setup-key sub-command: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: nbctl setup-key <list|show|create|revoke|delete>")
		os.Exit(1)
	}
}

func runSetupKeyList(args []string) {
	fs, cfg := newFlagSet("setup-key list")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	keys, err := listSetupKeys(cfg)
	must(err, "list setup keys")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, keys)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tSTATE\tUSED\tLIMIT\tEXPIRES\tEPHEMERAL")
	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%s\t%v\n",
			k.ID, k.Name, k.Type, dash(k.State), k.UsedTimes, k.UsageLimit, dash(k.ExpiresAt), k.Ephemeral)
	}
}

func runSetupKeyShow(args []string) {
	fs, cfg := newFlagSet("setup-key show")
	keyID := fs.String("id", "", "Setup key ID (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*keyID != "", "--id is required")

	k, err := getSetupKey(cfg, *keyID)
	must(err, "get setup key")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, k)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "ID\t%s\n", k.ID)
	fmt.Fprintf(w, "Key\t%s\n", k.Key)
	fmt.Fprintf(w, "Name\t%s\n", k.Name)
	fmt.Fprintf(w, "Type\t%s\n", k.Type)
	fmt.Fprintf(w, "State\t%s\n", dash(k.State))
	fmt.Fprintf(w, "Revoked\t%v\n", k.Revoked)
	fmt.Fprintf(w, "Ephemeral\t%v\n", k.Ephemeral)
	fmt.Fprintf(w, "UsedTimes\t%d\n", k.UsedTimes)
	fmt.Fprintf(w, "UsageLimit\t%d\n", k.UsageLimit)
	fmt.Fprintf(w, "ExpiresAt\t%s\n", dash(k.ExpiresAt))
	fmt.Fprintf(w, "LastUsed\t%s\n", dash(k.LastUsed))
	fmt.Fprintf(w, "CreatedAt\t%s\n", dash(k.CreatedAt))
	fmt.Fprintf(w, "UpdatedAt\t%s\n", dash(k.UpdatedAt))
}

func runSetupKeyCreate(args []string) {
	fs, cfg := newFlagSet("setup-key create")
	name := fs.String("name", "", "Setup key name (required)")
	keyType := fs.String("type", "one-off", "Key type: one-off or reusable")
	expiresIn := fs.Int("expires-in", 86400, "Expiry in seconds (86400–31536000)")
	ephemeral := fs.Bool("ephemeral", false, "Create ephemeral peers")
	usageLimit := fs.Int("usage-limit", 0, "Max uses (0 = unlimited)")
	var groups stringSliceFlag
	fs.Var(&groups, "group", "Auto-group ID (repeatable)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*name != "", "--name is required")

	reusable := *keyType == "reusable"
	k, err := createSetupKey(cfg, *name, *keyType, *expiresIn, reusable, *ephemeral, []string(groups), *usageLimit)
	must(err, "create setup key")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, k)
		return
	}
	logInfo("Key (shown once — store securely): %s", k.Key)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "ID\t%s\n", k.ID)
	fmt.Fprintf(w, "Name\t%s\n", k.Name)
	fmt.Fprintf(w, "Type\t%s\n", k.Type)
	fmt.Fprintf(w, "State\t%s\n", dash(k.State))
	fmt.Fprintf(w, "ExpiresAt\t%s\n", dash(k.ExpiresAt))
	fmt.Fprintf(w, "Ephemeral\t%v\n", k.Ephemeral)
}

func runSetupKeyRevoke(args []string) {
	fs, cfg := newFlagSet("setup-key revoke")
	keyID := fs.String("id", "", "Setup key ID to revoke (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*keyID != "", "--id is required")

	must(revokeSetupKey(cfg, *keyID), "revoke setup key")
	logInfo("Revoked setup key %s", *keyID)
}

func runSetupKeyDelete(args []string) {
	fs, cfg := newFlagSet("setup-key delete")
	keyID := fs.String("id", "", "Setup key ID to delete (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*keyID != "", "--id is required")

	must(deleteSetupKey(cfg, *keyID), "delete setup key")
	logInfo("Deleted setup key %s", *keyID)
}
