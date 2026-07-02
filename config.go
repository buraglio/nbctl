package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileConfig is the schema for an optional JSON or YAML config file.
type FileConfig struct {
	NetBirdURL   string `json:"netbird_url"   yaml:"netbird_url"`
	NetBirdToken string `json:"netbird_token" yaml:"netbird_token"`
}

// Config is the merged runtime configuration used across all commands.
type Config struct {
	NetBirdURL   string
	NetBirdToken string
	// Runtime
	Verbose    bool
	JSONOutput bool
	// Internal
	configFile string
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadConfigFile reads and parses a JSON or YAML config file.
func loadConfigFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var fc FileConfig
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
		if err := yaml.Unmarshal(data, &fc); err != nil {
			return nil, fmt.Errorf("parse YAML config: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &fc); err != nil {
			return nil, fmt.Errorf("parse JSON config: %w", err)
		}
	}
	return &fc, nil
}

// applyFileConfig merges file values into cfg, skipping flags set on the CLI.
func applyFileConfig(cfg *Config, fc *FileConfig, explicit map[string]bool) {
	if !explicit["netbird-url"] && fc.NetBirdURL != "" {
		cfg.NetBirdURL = fc.NetBirdURL
	}
	if !explicit["netbird-token"] && fc.NetBirdToken != "" {
		cfg.NetBirdToken = fc.NetBirdToken
	}
}

// addCommonFlags registers flags present on every sub-command.
func addCommonFlags(fs *flag.FlagSet, cfg *Config) {
	fs.StringVar(&cfg.NetBirdURL, "netbird-url", env("NETBIRD_URL", "https://api.netbird.io"), "NetBird management URL")
	fs.StringVar(&cfg.NetBirdToken, "netbird-token", env("NETBIRD_TOKEN", ""), "NetBird personal access token")
	fs.StringVar(&cfg.configFile, "config", "", "Path to JSON or YAML config file")
	fs.BoolVar(&cfg.Verbose, "v", false, "Verbose/debug output")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Verbose/debug output")
	fs.BoolVar(&cfg.JSONOutput, "json", false, "Output results as JSON")
}

// parseAndMerge parses args, then loads and merges any config file.
func parseAndMerge(fs *flag.FlagSet, cfg *Config, args []string) {
	fs.Parse(args)
	if cfg.configFile == "" {
		return
	}
	fc, err := loadConfigFile(cfg.configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
	explicit := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
	applyFileConfig(cfg, fc, explicit)
}
