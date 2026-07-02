package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// tagList is a repeatable string flag (--tag key:value, multiple allowed).
// NOTE: groups.go already defines stringSliceFlag for peer/group flags.
// tagList is a separate type used only for DNS record tags.
type tagList []string

func (t *tagList) String() string { return strings.Join(*t, ",") }
func (t *tagList) Set(v string) error {
	*t = append(*t, v)
	return nil
}

// bindNSList is a repeatable string flag for --bind-ns.
type bindNSList []string

func (b *bindNSList) String() string { return strings.Join(*b, ",") }
func (b *bindNSList) Set(v string) error {
	*b = append(*b, v)
	return nil
}

// ZoneTarget is a single Cloudflare zone to sync peers into.
// Tags are applied to every record in this zone (merged with global Tags).
type ZoneTarget struct {
	CFAPIToken string   `json:"cf_api_token" yaml:"cf_api_token"`
	CFZoneID   string   `json:"cf_zone_id"   yaml:"cf_zone_id"`
	Domain     string   `json:"domain"       yaml:"domain"`
	Tags       []string `json:"tags"         yaml:"tags"`
}

// FileConfig is the schema for an optional JSON or YAML config file.
// Pointer fields use *bool to distinguish "unset" from explicit false.
type FileConfig struct {
	// NetBird connection
	NetBirdURL   string `json:"netbird_url"   yaml:"netbird_url"`
	NetBirdToken string `json:"netbird_token" yaml:"netbird_token"`
	// Cloudflare single-zone shorthand
	CFAPIToken string `json:"cf_api_token" yaml:"cf_api_token"`
	CFZoneID   string `json:"cf_zone_id"   yaml:"cf_zone_id"`
	Domain     string `json:"domain"       yaml:"domain"`
	// Multi-zone
	Zones []ZoneTarget `json:"zones" yaml:"zones"`
	// Tag management
	ManagedTag  string   `json:"managed_tag"  yaml:"managed_tag"`
	Tags        []string `json:"tags"         yaml:"tags"`
	DisableTags bool     `json:"disable_tags" yaml:"disable_tags"`
	// Sync options
	TTL          int    `json:"ttl"           yaml:"ttl"`
	Proxied      bool   `json:"proxied"       yaml:"proxied"`
	Prune        bool   `json:"prune"         yaml:"prune"`
	DryRun       bool   `json:"dry_run"       yaml:"dry_run"`
	SyncIPv4     bool   `json:"sync_ipv4"     yaml:"sync_ipv4"`
	SyncIPv6     *bool  `json:"sync_ipv6"     yaml:"sync_ipv6"`
	ConnectedOnly bool   `json:"connected_only" yaml:"connected_only"`
	Comment      string `json:"comment"       yaml:"comment"`
	UseHostname  bool   `json:"use_hostname"  yaml:"use_hostname"`
	// BIND zone file
	BindZoneFile  string   `json:"bind_zone_file"  yaml:"bind_zone_file"`
	BindZoneDir   string   `json:"bind_zone_dir"   yaml:"bind_zone_dir"`
	BindNS        []string `json:"bind_ns"         yaml:"bind_ns"`
	BindSOAEmail  string   `json:"bind_soa_email"  yaml:"bind_soa_email"`
	BindReloadCmd string   `json:"bind_reload_cmd" yaml:"bind_reload_cmd"`
	BindFragment  bool     `json:"bind_fragment"   yaml:"bind_fragment"`
}

// Config is the merged runtime configuration used across all commands.
type Config struct {
	// NetBird connection
	NetBirdURL   string
	NetBirdToken string
	// Cloudflare single-zone (used when Zones is empty)
	CFAPIToken string
	CFZoneID   string
	Domain     string
	// Multi-zone (from config file zones array)
	Zones []ZoneTarget
	// Tag management
	ManagedTag  string
	Tags        tagList
	DisableTags bool
	// Sync options
	TTL           int
	Proxied       bool
	Prune         bool
	DryRun        bool
	SyncIPv4      bool
	SyncIPv6      bool
	ConnectedOnly bool
	Comment       string
	UseHostname   bool
	// BIND zone file
	BindZoneFile  string
	BindZoneDir   string
	BindNS        bindNSList
	BindSOAEmail  string
	BindReloadCmd string
	BindFragment  bool
	// Watch mode
	WatchInterval time.Duration
	// Runtime
	Verbose    bool
	JSONOutput bool
	// Internal
	configFile string
}

// effectiveTargets returns the ZoneTargets to use during sync.
// If Zones is configured (from config file), use those; otherwise build one
// from the single-zone flags.
func (cfg *Config) effectiveTargets() []ZoneTarget {
	if len(cfg.Zones) > 0 {
		return cfg.Zones
	}
	return []ZoneTarget{{
		CFAPIToken: cfg.CFAPIToken,
		CFZoneID:   cfg.CFZoneID,
		Domain:     cfg.Domain,
	}}
}

// buildTags assembles the full tag set for a record:
// [managed-tag] + global user tags + zone-specific tags (deduped).
func (cfg *Config) buildTags(zoneExtra []string) []string {
	seen := map[string]bool{}
	var tags []string
	add := func(t string) {
		if t != "" && !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}
	add(cfg.ManagedTag)
	for _, t := range cfg.Tags {
		add(t)
	}
	for _, t := range zoneExtra {
		add(t)
	}
	return tags
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
	str := func(name string, dst *string, src string) {
		if !explicit[name] && src != "" {
			*dst = src
		}
	}
	bl := func(name string, dst *bool, src bool) {
		if !explicit[name] && src {
			*dst = src
		}
	}
	// NetBird
	str("netbird-url", &cfg.NetBirdURL, fc.NetBirdURL)
	str("netbird-token", &cfg.NetBirdToken, fc.NetBirdToken)
	// Cloudflare single-zone
	str("cf-token", &cfg.CFAPIToken, fc.CFAPIToken)
	str("cf-zone", &cfg.CFZoneID, fc.CFZoneID)
	str("domain", &cfg.Domain, fc.Domain)
	// Multi-zone
	if len(fc.Zones) > 0 {
		cfg.Zones = fc.Zones
	}
	// Tags
	str("managed-tag", &cfg.ManagedTag, fc.ManagedTag)
	bl("disable-tags", &cfg.DisableTags, fc.DisableTags)
	if len(fc.Tags) > 0 && !explicit["tag"] {
		cfg.Tags = append(cfg.Tags, fc.Tags...)
	}
	// Sync options
	if !explicit["ttl"] && fc.TTL != 0 {
		cfg.TTL = fc.TTL
	}
	bl("proxied", &cfg.Proxied, fc.Proxied)
	bl("prune", &cfg.Prune, fc.Prune)
	bl("dry-run", &cfg.DryRun, fc.DryRun)
	bl("ipv4", &cfg.SyncIPv4, fc.SyncIPv4)
	bl("connected-only", &cfg.ConnectedOnly, fc.ConnectedOnly)
	bl("disable-tags", &cfg.DisableTags, fc.DisableTags)
	bl("use-hostname", &cfg.UseHostname, fc.UseHostname)
	str("comment", &cfg.Comment, fc.Comment)
	if !explicit["ipv6"] && fc.SyncIPv6 != nil {
		cfg.SyncIPv6 = *fc.SyncIPv6
	}
	// BIND
	str("bind-zone-file", &cfg.BindZoneFile, fc.BindZoneFile)
	str("bind-zone-dir", &cfg.BindZoneDir, fc.BindZoneDir)
	str("bind-soa-email", &cfg.BindSOAEmail, fc.BindSOAEmail)
	str("bind-reload-cmd", &cfg.BindReloadCmd, fc.BindReloadCmd)
	bl("bind-fragment", &cfg.BindFragment, fc.BindFragment)
	if len(fc.BindNS) > 0 && !explicit["bind-ns"] {
		cfg.BindNS = append(cfg.BindNS, fc.BindNS...)
	}
}

// addCommonFlags registers flags present on every sub-command.
func addCommonFlags(fs *flag.FlagSet, cfg *Config) {
	fs.StringVar(&cfg.NetBirdURL, "netbird-url", env("NETBIRD_URL", "https://api.netbird.io"), "NetBird management URL")
	fs.StringVar(&cfg.NetBirdToken, "netbird-token", env("NETBIRD_TOKEN", ""), "NetBird personal access token")
	fs.StringVar(&cfg.CFAPIToken, "cf-token", env("CLOUDFLARE_API_TOKEN", ""), "Cloudflare API token")
	fs.StringVar(&cfg.CFZoneID, "cf-zone", env("CLOUDFLARE_ZONE_ID", ""), "Cloudflare zone ID")
	fs.StringVar(&cfg.configFile, "config", "", "Path to JSON or YAML config file")
	fs.BoolVar(&cfg.Verbose, "v", false, "Verbose/debug output")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Verbose/debug output")
	fs.BoolVar(&cfg.JSONOutput, "json", false, "Output results as JSON")
}

// addSyncFlags registers flags for sync/zonefile/watch commands.
func addSyncFlags(fs *flag.FlagSet, cfg *Config) {
	fs.StringVar(&cfg.Domain, "domain", env("DOMAIN", ""), "Domain suffix (e.g. mesh.example.com)")
	fs.IntVar(&cfg.TTL, "ttl", 60, "DNS record TTL in seconds")
	fs.BoolVar(&cfg.Proxied, "proxied", false, "Proxy records through Cloudflare CDN")
	fs.BoolVar(&cfg.Prune, "prune", false, "Delete Cloudflare records absent from NetBird (managed-tag only)")
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "Preview changes without applying")
	fs.BoolVar(&cfg.SyncIPv4, "ipv4", false, "Sync A (IPv4) records; if neither --ipv4 nor --ipv6 is set, both are synced")
	fs.BoolVar(&cfg.SyncIPv6, "ipv6", false, "Sync AAAA (IPv6) records; if neither --ipv4 nor --ipv6 is set, both are synced")
	fs.BoolVar(&cfg.ConnectedOnly, "connected-only", false, "Skip peers that are not currently connected")
	fs.StringVar(&cfg.Comment, "comment", "Managed by nbctl", "Comment written to every DNS record")
	fs.StringVar(&cfg.ManagedTag, "managed-tag", "managed:nbctl", "Tag stamped on every managed record; prune only removes records with this tag")
	fs.Var(&cfg.Tags, "tag", "Extra tag to apply to every record (key:value, repeatable)")
	fs.BoolVar(&cfg.DisableTags, "disable-tags", false, "Omit tags from Cloudflare API calls (required for free/non-Enterprise zones)")
	fs.BoolVar(&cfg.UseHostname, "use-hostname", false, "Use the machine hostname instead of the NetBird DNS label for DNS records")
	// BIND zone file flags
	fs.StringVar(&cfg.BindZoneFile, "bind-zone-file", "", "Write BIND zone file to this path (stdout if empty)")
	fs.StringVar(&cfg.BindZoneDir, "bind-zone-dir", "", "Write one BIND zone file per zone to this directory")
	fs.Var(&cfg.BindNS, "bind-ns", "NS record for generated zone (repeatable)")
	fs.StringVar(&cfg.BindSOAEmail, "bind-soa-email", "", "SOA RNAME (default: hostmaster.<domain>.)")
	fs.StringVar(&cfg.BindReloadCmd, "bind-reload-cmd", "", "Shell command run after each zone file write")
	fs.BoolVar(&cfg.BindFragment, "bind-fragment", false, "Write only A/AAAA records — no SOA/NS header")
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
