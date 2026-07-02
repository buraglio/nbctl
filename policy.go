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

// NetBirdPolicyRule is a single traffic-matching rule within a policy.
type NetBirdPolicyRule struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Enabled       bool     `json:"enabled"`
	Action        string   `json:"action"`       // "accept" or "drop"
	Bidirectional bool     `json:"bidirectional"`
	Protocol      string   `json:"protocol"`     // "all", "tcp", "udp", "icmp"
	Ports         []string `json:"ports,omitempty"`
	Sources       []string `json:"sources"`
	Destinations  []string `json:"destinations"`
}

// NetBirdPolicy is an access-control policy containing one or more rules.
type NetBirdPolicy struct {
	ID                  string              `json:"id,omitempty"`
	Name                string              `json:"name"`
	Description         string              `json:"description,omitempty"`
	Enabled             bool                `json:"enabled"`
	Rules               []NetBirdPolicyRule `json:"rules"`
	SourcePostureChecks []string            `json:"source_posture_checks,omitempty"`
}

func nbPolicyClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }

// listPolicies returns all access policies via GET /api/policies.
func listPolicies(cfg *Config) ([]NetBirdPolicy, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/policies", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbPolicyClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result []NetBirdPolicy
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode policies response: %w", err)
	}
	return result, nil
}

// getPolicy fetches a single policy by ID via GET /api/policies/{policyID}.
func getPolicy(cfg *Config, policyID string) (NetBirdPolicy, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/policies/"+policyID, nil)
	if err != nil {
		return NetBirdPolicy{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbPolicyClient().Do(req)
	if err != nil {
		return NetBirdPolicy{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdPolicy{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result NetBirdPolicy
	if err := json.Unmarshal(body, &result); err != nil {
		return NetBirdPolicy{}, fmt.Errorf("decode policy response: %w", err)
	}
	return result, nil
}

// deletePolicy removes a policy via DELETE /api/policies/{policyID}.
func deletePolicy(cfg *Config, policyID string) error {
	req, err := http.NewRequest(http.MethodDelete, cfg.NetBirdURL+"/api/policies/"+policyID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbPolicyClient().Do(req)
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

// createPolicy creates a new policy via POST /api/policies.
func createPolicy(cfg *Config, p NetBirdPolicy) (NetBirdPolicy, error) {
	encoded, err := json.Marshal(p)
	if err != nil {
		return NetBirdPolicy{}, err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.NetBirdURL+"/api/policies", bytes.NewReader(encoded))
	if err != nil {
		return NetBirdPolicy{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbPolicyClient().Do(req)
	if err != nil {
		return NetBirdPolicy{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdPolicy{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result NetBirdPolicy
	if err := json.Unmarshal(body, &result); err != nil {
		return NetBirdPolicy{}, fmt.Errorf("decode create-policy response: %w", err)
	}
	return result, nil
}

// updatePolicy replaces a policy via PUT /api/policies/{policyID}.
func updatePolicy(cfg *Config, policyID string, p NetBirdPolicy) (NetBirdPolicy, error) {
	encoded, err := json.Marshal(p)
	if err != nil {
		return NetBirdPolicy{}, err
	}

	req, err := http.NewRequest(http.MethodPut, cfg.NetBirdURL+"/api/policies/"+policyID, bytes.NewReader(encoded))
	if err != nil {
		return NetBirdPolicy{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbPolicyClient().Do(req)
	if err != nil {
		return NetBirdPolicy{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdPolicy{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var result NetBirdPolicy
	if err := json.Unmarshal(body, &result); err != nil {
		return NetBirdPolicy{}, fmt.Errorf("decode update-policy response: %w", err)
	}
	return result, nil
}

// ── policy command ─────────────────────────────────────────────────────────────

func runPolicy(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl policy <list|show|create|delete> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		runPolicyList(rest)
	case "show":
		runPolicyShow(rest)
	case "create":
		runPolicyCreate(rest)
	case "delete":
		runPolicyDelete(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown policy sub-command: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: nbctl policy <list|show|create|delete>")
		os.Exit(1)
	}
}

func runPolicyList(args []string) {
	fs, cfg := newFlagSet("policy list")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	policies, err := listPolicies(cfg)
	must(err, "list policies")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, policies)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "ID\tNAME\tENABLED\tRULES")
	for _, p := range policies {
		fmt.Fprintf(w, "%s\t%s\t%v\t%d\n", p.ID, p.Name, p.Enabled, len(p.Rules))
	}
}

func runPolicyShow(args []string) {
	fs, cfg := newFlagSet("policy show")
	policyID := fs.String("id", "", "Policy ID (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*policyID != "", "--id is required")

	p, err := getPolicy(cfg, *policyID)
	must(err, "get policy")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, p)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\t%s\n", p.ID)
	fmt.Fprintf(w, "Name\t%s\n", p.Name)
	fmt.Fprintf(w, "Description\t%s\n", dash(p.Description))
	fmt.Fprintf(w, "Enabled\t%v\n", p.Enabled)
	fmt.Fprintf(w, "Rules\t%d\n", len(p.Rules))
	w.Flush()

	if len(p.Rules) > 0 {
		fmt.Println()
		rw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(rw, "NAME\tACTION\tPROTO\tBIDIRECTIONAL\tSOURCES\tDESTINATIONS")
		for _, rule := range p.Rules {
			fmt.Fprintf(rw, "%s\t%s\t%s\t%v\t%s\t%s\n",
				rule.Name,
				rule.Action,
				rule.Protocol,
				rule.Bidirectional,
				strings.Join(rule.Sources, ","),
				strings.Join(rule.Destinations, ","),
			)
		}
		rw.Flush()
	}
}

func runPolicyCreate(args []string) {
	fs, cfg := newFlagSet("policy create")
	name := fs.String("name", "", "Policy name (required)")
	description := fs.String("description", "", "Policy description")
	ruleName := fs.String("rule-name", "", "Rule name (defaults to policy name)")
	ruleAction := fs.String("rule-action", "accept", "Rule action: accept or drop")
	ruleProto := fs.String("rule-proto", "all", "Rule protocol: all, tcp, udp, icmp")
	bidirectional := fs.Bool("bidirectional", true, "Rule is bidirectional")
	enabled := fs.Bool("enabled", true, "Policy and rule are enabled")
	var rulePorts stringSliceFlag
	fs.Var(&rulePorts, "rule-port", "Port to match (repeatable)")
	var ruleSrcs stringSliceFlag
	fs.Var(&ruleSrcs, "rule-src", "Source group ID (repeatable)")
	var ruleDsts stringSliceFlag
	fs.Var(&ruleDsts, "rule-dst", "Destination group ID (repeatable)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*name != "", "--name is required")

	rname := *ruleName
	if rname == "" {
		rname = *name
	}

	rule := NetBirdPolicyRule{
		Name:          rname,
		Enabled:       *enabled,
		Action:        *ruleAction,
		Bidirectional: *bidirectional,
		Protocol:      *ruleProto,
		Sources:       nonNilSlice([]string(ruleSrcs)),
		Destinations:  nonNilSlice([]string(ruleDsts)),
	}
	if len(rulePorts) > 0 {
		rule.Ports = []string(rulePorts)
	}

	p := NetBirdPolicy{
		Name:        *name,
		Description: *description,
		Enabled:     *enabled,
		Rules:       []NetBirdPolicyRule{rule},
	}

	created, err := createPolicy(cfg, p)
	must(err, "create policy")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, created)
		return
	}
	logInfo("Created policy %q (id %s)", created.Name, created.ID)
}

func runPolicyDelete(args []string) {
	fs, cfg := newFlagSet("policy delete")
	policyID := fs.String("id", "", "Policy ID to delete (required)")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*policyID != "", "--id is required")

	must(deletePolicy(cfg, *policyID), "delete policy")
	logInfo("Deleted policy %s", *policyID)
}

// nonNilSlice returns an empty slice instead of nil so JSON serialises to [].
func nonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

