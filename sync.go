package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// SyncStats tracks what happened during a single sync run.
type SyncStats struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Deleted   int `json:"deleted"`
	Errors    int `json:"errors"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

// peerDNSLabel returns the short label (no domain) for DNS record names.
// NetBird's dns_label is the full FQDN (e.g. "peer.domain.com"), so we
// take only the first segment. With --use-hostname we use the machine
// hostname instead.
func peerDNSLabel(p NetBirdPeer, useHostname bool) string {
	if useHostname {
		if p.Hostname != "" {
			return strings.SplitN(p.Hostname, ".", 2)[0]
		}
		return p.Name
	}
	if p.DNSLabel != "" {
		return strings.SplitN(p.DNSLabel, ".", 2)[0]
	}
	return p.Name
}

// peerIP returns the peer's IP for the requested record type, or "".
// NetBird exposes separate fields: ip (IPv4) and ipv6 (IPv6, read-only).
func peerIP(p NetBirdPeer, recType string) string {
	switch recType {
	case "A":
		return p.IP
	case "AAAA":
		return p.IPv6
	}
	return ""
}

// filterPeers returns peers that pass the ConnectedOnly filter.
func filterPeers(peers []NetBirdPeer, connectedOnly bool) []NetBirdPeer {
	if !connectedOnly {
		return peers
	}
	out := peers[:0:0]
	for _, p := range peers {
		if p.Connected {
			out = append(out, p)
		}
	}
	return out
}

// ── sync command ──────────────────────────────────────────────────────────────

func runSync(args []string) {
	fs, cfg := newFlagSet("sync")
	addSyncFlags(fs, cfg)
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(cfg.ManagedTag != "", "managed-tag must not be empty")
	if len(cfg.Zones) == 0 {
		require(cfg.CFAPIToken != "", "cf-token is required (or set zones in config file)")
		require(cfg.CFZoneID != "", "cf-zone is required (or set zones in config file)")
		require(cfg.Domain != "", "domain is required (or set zones in config file)")
	}

	stats, err := doSync(cfg)
	must(err, "sync")

	if cfg.JSONOutput {
		mustEncodeJSON(os.Stdout, stats)
		return
	}
	if cfg.DryRun {
		logInfo("DRY RUN complete — no changes made")
	} else {
		logInfo("Sync complete: %d created, %d updated, %d unchanged, %d deleted, %d errors",
			stats.Created, stats.Updated, stats.Unchanged, stats.Deleted, stats.Errors)
	}
}

// ── watch command ─────────────────────────────────────────────────────────────

func runWatch(args []string) {
	fs, cfg := newFlagSet("watch")
	addSyncFlags(fs, cfg)
	fs.DurationVar(&cfg.WatchInterval, "interval", 5*time.Minute, "Sync interval (e.g. 30s, 5m, 1h)")
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(cfg.ManagedTag != "", "managed-tag must not be empty")
	if len(cfg.Zones) == 0 {
		require(cfg.CFAPIToken != "", "cf-token is required")
		require(cfg.CFZoneID != "", "cf-zone is required")
		require(cfg.Domain != "", "domain is required")
	}

	logInfo("Watch mode: syncing every %s", cfg.WatchInterval)
	doSyncLogged(cfg)
	ticker := time.NewTicker(cfg.WatchInterval)
	defer ticker.Stop()
	for range ticker.C {
		doSyncLogged(cfg)
	}
}

func doSyncLogged(cfg *Config) {
	start := time.Now()
	stats, err := doSync(cfg)
	dur := time.Since(start)
	if err != nil {
		logWarn("Sync error: %v", err)
		return
	}
	logInfo("Sync done in %s: %d created, %d updated, %d unchanged, %d deleted, %d errors",
		dur.Round(time.Millisecond),
		stats.Created, stats.Updated, stats.Unchanged, stats.Deleted, stats.Errors)
}

// ── core sync logic ───────────────────────────────────────────────────────────

// doSync fetches NetBird peers and syncs them to every configured Cloudflare zone.
func doSync(cfg *Config) (*SyncStats, error) {
	if cfg.DryRun {
		logInfo("DRY RUN — no changes will be made")
	}

	peers, err := fetchPeers(cfg)
	if err != nil {
		return nil, fmt.Errorf("fetch peers: %w", err)
	}
	logInfo("Found %d NetBird peers", len(peers))

	filtered := filterPeers(peers, cfg.ConnectedOnly)
	if cfg.ConnectedOnly {
		logInfo("After connected-only filter: %d peers", len(filtered))
	}

	// Default: sync both when neither flag is explicitly set.
	syncV4, syncV6 := cfg.SyncIPv4, cfg.SyncIPv6
	if !syncV4 && !syncV6 {
		syncV4, syncV6 = true, true
	}

	stats := &SyncStats{}

	for _, target := range cfg.effectiveTargets() {
		logInfo("Zone %s: syncing %d peers", target.Domain, len(filtered))

		if syncV4 {
			if err := syncRecordType(cfg, filtered, "A", target, stats); err != nil {
				return stats, fmt.Errorf("zone %s A: %w", target.Domain, err)
			}
		}
		if syncV6 {
			if err := syncRecordType(cfg, filtered, "AAAA", target, stats); err != nil {
				return stats, fmt.Errorf("zone %s AAAA: %w", target.Domain, err)
			}
		}
	}
	return stats, nil
}

func syncRecordType(cfg *Config, peers []NetBirdPeer, recType string, target ZoneTarget, stats *SyncStats) error {
	existing, err := fetchCFRecords(cfg, target.CFAPIToken, target.CFZoneID, recType)
	if err != nil {
		return fmt.Errorf("fetch %s records: %w", recType, err)
	}
	logInfo("  %s: %d existing records in Cloudflare", recType, len(existing))

	byName := make(map[string]CloudflareRecord, len(existing))
	for _, r := range existing {
		byName[r.Name] = r
	}

	wantedTags := cfg.buildTags(target.Tags)
	nbFQDNs := make(map[string]bool)
	withAddr := 0

	for _, p := range peers {
		ip := peerIP(p, recType)
		if ip == "" {
			continue
		}
		withAddr++

		fqdn := peerDNSLabel(p, cfg.UseHostname) + "." + target.Domain
		nbFQDNs[fqdn] = true

		rec, exists := byName[fqdn]
		if !exists {
			logInfo("  CREATE %s %s %s", recType, fqdn, ip)
			if !cfg.DryRun {
				if err := createCFRecord(cfg, target.CFAPIToken, target.CFZoneID, fqdn, ip, recType, wantedTags); err != nil {
					logWarn("    create failed: %v", err)
					stats.Errors++
				} else {
					stats.Created++
				}
			}
			continue
		}

		ipChanged := rec.Content != ip
		tagsChanged := !cfg.DisableTags && !tagsMatch(rec.Tags, wantedTags)
		if ipChanged || tagsChanged {
			if ipChanged {
				logInfo("  UPDATE %s %s  %s -> %s", recType, fqdn, rec.Content, ip)
			} else {
				logInfo("  UPDATE %s %s (tags changed)", recType, fqdn)
			}
			if !cfg.DryRun {
				if err := updateCFRecord(cfg, target.CFAPIToken, target.CFZoneID, rec.ID, fqdn, ip, recType, wantedTags); err != nil {
					logWarn("    update failed: %v", err)
					stats.Errors++
				} else {
					stats.Updated++
				}
			}
		} else {
			logDebug(cfg, "UNCHANGED %s %s (%s)", recType, fqdn, ip)
			stats.Unchanged++
		}
	}

	if withAddr == 0 {
		logInfo("  %s: no peers have %s addresses — nothing to sync", recType, recType)
	}

	if cfg.Prune {
		suffix := "." + target.Domain
		for name, rec := range byName {
			if !strings.HasSuffix(name, suffix) {
				continue
			}
			if !hasManagedTag(rec, cfg.ManagedTag) {
				logDebug(cfg, "SKIP prune %s: no managed tag %q", name, cfg.ManagedTag)
				continue
			}
			if !nbFQDNs[name] {
				logInfo("  DELETE %s %s (%s) — not in NetBird", recType, name, rec.Content)
				if !cfg.DryRun {
					if err := deleteCFRecord(cfg, target.CFAPIToken, target.CFZoneID, rec.ID); err != nil {
						logWarn("    delete failed: %v", err)
						stats.Errors++
					} else {
						stats.Deleted++
					}
				}
			}
		}
	}
	return nil
}

// ── zonefile command ──────────────────────────────────────────────────────────

func runZonefile(args []string) {
	fs, cfg := newFlagSet("zonefile")
	addSyncFlags(fs, cfg)
	parseAndMerge(fs, cfg, args)

	require(cfg.NetBirdToken != "", "netbird-token is required")
	require(cfg.Domain != "", "domain is required")

	peers, err := fetchPeers(cfg)
	must(err, "fetch peers")

	filtered := filterPeers(peers, cfg.ConnectedOnly)

	if len(cfg.Zones) > 0 {
		// Multi-zone: write one file per zone
		require(cfg.BindZoneDir != "", "--bind-zone-dir is required for multi-zone output")
		for _, target := range cfg.Zones {
			outPath := filepath.Join(cfg.BindZoneDir, target.Domain+".zone")
			f, err := os.Create(outPath)
			must(err, "create zone file "+outPath)
			must(writeBindZone(cfg, filtered, f, target.Domain), "write zone "+target.Domain)
			f.Close()
			logInfo("Wrote %s", outPath)
			if cfg.BindReloadCmd != "" {
				runReloadCmd(cfg.BindReloadCmd)
			}
		}
		return
	}

	// Single-zone
	if cfg.BindZoneFile != "" {
		f, err := os.Create(cfg.BindZoneFile)
		must(err, "create zone file")
		must(writeBindZone(cfg, filtered, f, cfg.Domain), "write zone")
		f.Close()
		logInfo("Wrote %s", cfg.BindZoneFile)
		if cfg.BindReloadCmd != "" {
			runReloadCmd(cfg.BindReloadCmd)
		}
		return
	}

	// stdout
	must(writeBindZone(cfg, filtered, os.Stdout, cfg.Domain), "write zone")
}

// writeBindZone writes a BIND-format zone file for the given domain.
func writeBindZone(cfg *Config, peers []NetBirdPeer, w io.Writer, domain string) error {
	now := time.Now().UTC()
	serial := now.Unix()

	if !cfg.BindFragment {
		soaEmail := cfg.BindSOAEmail
		if soaEmail == "" {
			soaEmail = "hostmaster." + domain + "."
		}

		ns1 := "ns1." + domain + "."
		if len(cfg.BindNS) > 0 {
			ns1 = cfg.BindNS[0]
		}

		fmt.Fprintf(w, "; Generated by nbctl %s — %s\n", version, now.Format(time.RFC3339))
		fmt.Fprintf(w, "$ORIGIN %s.\n", domain)
		fmt.Fprintf(w, "$TTL %d\n\n", cfg.TTL)
		fmt.Fprintf(w, "@  IN SOA  %s %s (\n", ns1, soaEmail)
		fmt.Fprintf(w, "               %d ; serial\n", serial)
		fmt.Fprintf(w, "               3600       ; refresh\n")
		fmt.Fprintf(w, "               900        ; retry\n")
		fmt.Fprintf(w, "               604800     ; expire\n")
		fmt.Fprintf(w, "               %d         ; minimum\n", cfg.TTL)
		fmt.Fprintf(w, "           )\n\n")

		for _, ns := range cfg.BindNS {
			fmt.Fprintf(w, "@  IN NS   %s\n", ns)
		}
		if len(cfg.BindNS) > 0 {
			fmt.Fprintln(w)
		}
	}

	// Default: emit both when neither flag is explicitly set.
	writeV4, writeV6 := cfg.SyncIPv4, cfg.SyncIPv6
	if !writeV4 && !writeV6 {
		writeV4, writeV6 = true, true
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// A records
	if writeV4 {
		wrote := false
		for _, p := range peers {
			ip := peerIP(p, "A")
			if ip == "" {
				continue
			}
			label := peerDNSLabel(p, cfg.UseHostname)
			fmt.Fprintf(tw, "%s\tIN A\t%s\n", label, ip)
			wrote = true
		}
		tw.Flush()
		if !wrote {
			fmt.Fprintf(w, "; no peers have A (IPv4) addresses\n")
		}
	}

	// AAAA records
	if writeV6 {
		wrote := false
		for _, p := range peers {
			ip := peerIP(p, "AAAA")
			if ip == "" {
				continue
			}
			label := peerDNSLabel(p, cfg.UseHostname)
			fmt.Fprintf(tw, "%s\tIN AAAA\t%s\n", label, ip)
			wrote = true
		}
		tw.Flush()
		if !wrote {
			fmt.Fprintf(w, "; no peers have AAAA (IPv6) addresses\n")
		}
	}

	return nil
}

func runReloadCmd(cmd string) {
	logInfo("Running: %s", cmd)
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		logWarn("reload command failed: %v\n%s", err, out)
	} else {
		logInfo("reload command succeeded")
	}
}
