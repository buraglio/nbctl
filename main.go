package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

// Build-time default. Override at compile time:
//
//	go build -ldflags="-X main.defaultNetBirdURL=https://api.netbird.io" .
var defaultNetBirdURL = "https://api.netbird.io"

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "peers":
		runPeers(os.Args[2:])
	case "groups":
		runGroups(os.Args[2:])
	case "users":
		runUsers(os.Args[2:])
	case "setup-key":
		runSetupKey(os.Args[2:])
	case "routes":
		runRoutes(os.Args[2:])
	case "policy":
		runPolicy(os.Args[2:])
	case "events":
		runEvents(os.Args[2:])
	case "posture-check":
		runPostureCheck(os.Args[2:])
	case "completion":
		runCompletion(os.Args[2:])
	case "version", "--version", "-version":
		fmt.Printf("nbctl %s\n", version)
	case "help", "--help", "-help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `nbctl — NetBird management CLI

Usage:
  nbctl <command> [flags]

Commands:
  peers          Manage peers (sub-commands: list, show, update, delete)
  groups         Manage groups (sub-commands: list, show, create, update, delete)
  users          Manage users (sub-commands: list, create, delete, invite, approve)
  setup-key      Manage setup keys (sub-commands: list, show, create, update, delete)
  routes         Manage network routes (sub-commands: list, show, create, update, delete)
  policy         Manage access policies (sub-commands: list, show, create, update, delete)
  events         View audit and network traffic events (sub-commands: audit, traffic)
  posture-check  Manage posture checks (sub-commands: list, show, create, update, delete)
  completion     Generate shell completion scripts (bash, zsh, fish)
  version        Print version information

Run 'nbctl <command> -help' for command-specific flags.

Environment variables (all commands):
  NETBIRD_URL     NetBird management URL (default: https://api.netbird.io)
  NETBIRD_TOKEN   NetBird personal access token

Config file (JSON or YAML, pass with --config):
  {
    "netbird_url":   "https://api.netbird.io",
    "netbird_token": "nbp_..."
  }

Compiled-in default URL (set at build time for self-hosted):
  go build -ldflags="-X main.defaultNetBirdURL=https://nb.example.com" .
`)
}

// ── shared helpers ─────────────────────────────────────────────────────────────

func logInfo(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[INFO]  "+format+"\n", args...)
}

func logDebug(cfg *Config, format string, args ...interface{}) {
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func require(cond bool, msg string) {
	if !cond {
		fmt.Fprintln(os.Stderr, "[ERROR] "+msg)
		os.Exit(1)
	}
}

func must(err error, context string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", context, err)
		os.Exit(1)
	}
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func mustEncodeJSON(w io.Writer, v interface{}) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		must(err, "encode JSON")
	}
}

func newFlagSet(name string) (*flag.FlagSet, *Config) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	cfg := &Config{}
	addCommonFlags(fs, cfg)
	return fs, cfg
}
