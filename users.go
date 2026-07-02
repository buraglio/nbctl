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

// NetBirdUser represents a user in the NetBird management API.
type NetBirdUser struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Role          string   `json:"role"`
	Status        string   `json:"status"`
	LastLogin     string   `json:"last_login"`
	AutoGroups    []string `json:"auto_groups"`
	IsServiceUser bool     `json:"is_service_user"`
	IsBlocked     bool     `json:"is_blocked"`
}

func nbUserClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }

// listUsers fetches all users via GET /api/users.
func listUsers(cfg *Config) ([]NetBirdUser, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.NetBirdURL+"/api/users", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Accept", "application/json")

	resp, err := nbUserClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, body)
	}
	var users []NetBirdUser
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("decode users response: %w", err)
	}
	return users, nil
}

// createUser creates a new user via POST /api/users.
func createUser(cfg *Config, email, name, role string, isServiceUser bool, autoGroups []string) (NetBirdUser, error) {
	if autoGroups == nil {
		autoGroups = []string{}
	}
	payload := map[string]interface{}{
		"email":           email,
		"name":            name,
		"role":            role,
		"is_service_user": isServiceUser,
		"auto_groups":     autoGroups,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return NetBirdUser{}, err
	}
	req, err := http.NewRequest(http.MethodPost, cfg.NetBirdURL+"/api/users", bytes.NewReader(body))
	if err != nil {
		return NetBirdUser{}, err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := nbUserClient().Do(req)
	if err != nil {
		return NetBirdUser{}, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return NetBirdUser{}, fmt.Errorf("NetBird API HTTP %d: %s", resp.StatusCode, respBody)
	}
	var u NetBirdUser
	if err := json.Unmarshal(respBody, &u); err != nil {
		return NetBirdUser{}, fmt.Errorf("decode create-user response: %w", err)
	}
	return u, nil
}

// deleteUser removes a user by ID via DELETE /api/users/{userID}.
func deleteUser(cfg *Config, userID string) error {
	req, err := http.NewRequest(http.MethodDelete, cfg.NetBirdURL+"/api/users/"+userID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbUserClient().Do(req)
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

// inviteUser sends an invitation to a user via POST /api/users/{userID}/invite.
func inviteUser(cfg *Config, userID string) error {
	req, err := http.NewRequest(http.MethodPost, cfg.NetBirdURL+"/api/users/"+userID+"/invite", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbUserClient().Do(req)
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

// approveUser approves a pending user via POST /api/users/{userID}/approve.
func approveUser(cfg *Config, userID string) error {
	req, err := http.NewRequest(http.MethodPost, cfg.NetBirdURL+"/api/users/"+userID+"/approve", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+cfg.NetBirdToken)

	resp, err := nbUserClient().Do(req)
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

// findUserByEmail returns the first user whose Email matches exactly.
func findUserByEmail(users []NetBirdUser, email string) (NetBirdUser, error) {
	for _, u := range users {
		if u.Email == email {
			return u, nil
		}
	}
	return NetBirdUser{}, fmt.Errorf("no user found with email %q", email)
}

// ── users command ─────────────────────────────────────────────────────────────

func runUsers(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl users <list|create|delete|invite|approve> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		runUsersList(rest)
	case "create":
		runUsersCreate(rest)
	case "delete":
		runUsersDelete(rest)
	case "invite":
		runUsersInvite(rest)
	case "approve":
		runUsersApprove(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown users sub-command: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: nbctl users <list|create|delete|invite|approve>")
		os.Exit(1)
	}
}

func runUsersList(args []string) {
	fs, cfg := newFlagSet("users list")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")

	users, err := listUsers(cfg)
	must(err, "list users")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, users)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "ID\tEMAIL\tNAME\tROLE\tSTATUS\tSERVICE")
	for _, u := range users {
		svc := "no"
		if u.IsServiceUser {
			svc = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", u.ID, u.Email, u.Name, u.Role, u.Status, svc)
	}
}

func runUsersCreate(args []string) {
	fs, cfg := newFlagSet("users create")
	email := fs.String("email", "", "User email address")
	name := fs.String("name", "", "User display name")
	role := fs.String("role", "user", "User role (user, admin)")
	service := fs.Bool("service", false, "Create as service user")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	if *service {
		require(*name != "", "--name is required for service users")
	} else {
		require(*email != "", "--email is required")
	}

	u, err := createUser(cfg, *email, *name, *role, *service, nil)
	must(err, "create user")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, u)
		return
	}
	logInfo("Created user %q (id %s)", u.Email, u.ID)
}

func runUsersDelete(args []string) {
	fs, cfg := newFlagSet("users delete")
	id := fs.String("id", "", "User ID to delete")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*id != "", "--id is required")

	must(deleteUser(cfg, *id), "delete user")
	logInfo("Deleted user %q", *id)
}

func runUsersInvite(args []string) {
	fs, cfg := newFlagSet("users invite")
	id := fs.String("id", "", "User ID to invite")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*id != "", "--id is required")

	must(inviteUser(cfg, *id), "invite user")
	logInfo("Invited user %q", *id)
}

func runUsersApprove(args []string) {
	fs, cfg := newFlagSet("users approve")
	id := fs.String("id", "", "User ID to approve")
	parseAndMerge(fs, cfg, args)
	require(cfg.NetBirdURL != "", "netbird-url is required")
	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(*id != "", "--id is required")

	must(approveUser(cfg, *id), "approve user")
	logInfo("Approved user %q", *id)
}
