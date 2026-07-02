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

// NetBirdPostureCheck represents a posture check policy from the NetBird API.
type NetBirdPostureCheck struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Checks      struct {
		NBVersionCheck *struct {
			MinVersion string `json:"min_version"`
		} `json:"nb_version_check,omitempty"`
		OSVersionCheck *struct {
			Linux   *struct{ MinKernelVersion string `json:"min_kernel_version"` } `json:"linux,omitempty"`
			Darwin  *struct{ MinVersion string `json:"min_version"` } `json:"darwin,omitempty"`
			Windows *struct{ MinKernelVersion string `json:"min_kernel_version"` } `json:"windows,omitempty"`
		} `json:"os_version_check,omitempty"`
		GeoLocationCheck *struct {
			Locations []struct {
				CountryCode string `json:"country_code"`
				CityName    string `json:"city_name,omitempty"`
			} `json:"locations"`
			Action string `json:"action"` // "allow" or "deny"
		} `json:"geo_location_check,omitempty"`
	} `json:"checks"`
}

func nbPostureClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }

// listPostureChecks returns all posture checks via GET /api/posture-checks.
func listPostureChecks(cfg *Config) ([]NetBirdPostureCheck, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/posture-checks", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbPostureClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, b)
	}
	var result []NetBirdPostureCheck
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode posture-checks response: %w", err)
	}
	return result, nil
}

// getPostureCheck fetches a single posture check via GET /api/posture-checks/{checkID}.
func getPostureCheck(cfg *Config, checkID string) (NetBirdPostureCheck, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/posture-checks/"+checkID, nil)
	if err != nil {
		return NetBirdPostureCheck{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbPostureClient().Do(req)
	if err != nil {
		return NetBirdPostureCheck{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return NetBirdPostureCheck{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, b)
	}
	var result NetBirdPostureCheck
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return NetBirdPostureCheck{}, fmt.Errorf("decode posture-check response: %w", err)
	}
	return result, nil
}

// deletePostureCheck removes a posture check via DELETE /api/posture-checks/{checkID}.
func deletePostureCheck(cfg *Config, checkID string) error {
	req, err := http.NewRequest(http.MethodDelete, cfg.NetBirdURL+"/api/posture-checks/"+checkID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbPostureClient().Do(req)
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

// ── posture-check command ─────────────────────────────────────────────────────

func runPostureCheck(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl posture-check <list|show|delete> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		runPostureCheckList(rest)
	case "show":
		runPostureCheckShow(rest)
	case "delete":
		runPostureCheckDelete(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown posture-check sub-command: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: nbctl posture-check <list|show|delete>")
		os.Exit(1)
	}
}

func runPostureCheckList(args []string) {
	fs, cfg := newFlagSet("posture-check list")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	checks, err := listPostureChecks(cfg)
	must(err, "list posture checks")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, checks)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
	for _, c := range checks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, c.Name, dash(c.Description))
	}
}

func runPostureCheckShow(args []string) {
	fs, cfg := newFlagSet("posture-check show")
	id := fs.String("id", "", "Posture check ID")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*id != "", "--id is required")

	c, err := getPostureCheck(cfg, *id)
	must(err, "get posture check")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, c)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "ID\t%s\n", c.ID)
	fmt.Fprintf(w, "Name\t%s\n", c.Name)
	fmt.Fprintf(w, "Description\t%s\n", dash(c.Description))
}

func runPostureCheckDelete(args []string) {
	fs, cfg := newFlagSet("posture-check delete")
	id := fs.String("id", "", "Posture check ID to delete")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*id != "", "--id is required")

	must(deletePostureCheck(cfg, *id), "delete posture check")
	logInfo("Deleted posture check %s", *id)
}
