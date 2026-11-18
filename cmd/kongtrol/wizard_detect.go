package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

// ── VPN client detection ──────────────────────────────────────────────────────
//
// Where each client lives per OS is shared with the connect preflight and
// doctor in internal/vpndetect; this file only adds the wizard's version label.

type detectedVPN struct {
	label       string
	version     string
	adapterType string // "" when Kongtrol has no adapter for the client
	path        string // detected binary, offered as binary_path default
}

func detectInstalledVPNs() []detectedVPN {
	clients := vpndetect.DetectAll()
	found := make([]detectedVPN, 0, len(clients))
	for _, c := range clients {
		found = append(found, detectedVPN{
			label:       c.Label,
			version:     versionOf(c.Path, c.VersionArgs),
			adapterType: c.AdapterType,
			path:        c.Path,
		})
	}
	return found
}

func versionOf(path string, args []string) string {
	if len(args) > 0 {
		if v := runVersion(path, args); v != "" {
			return v
		}
	}
	if v := vpndetect.FileVersion(path); v != "" {
		return v
	}
	parent := filepath.Base(filepath.Dir(path))
	if strings.HasPrefix(parent, "v") && len(parent) > 1 {
		return parent
	}
	grandparent := filepath.Base(filepath.Dir(filepath.Dir(path)))
	if strings.HasPrefix(grandparent, "v") && len(grandparent) > 1 {
		return grandparent
	}
	return "installed"
}

func runVersion(path string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, args...).Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".dll") ||
			strings.HasSuffix(lower, ".node") || strings.HasPrefix(lower, "loaded ") {
			continue
		}
		if len(line) > 60 {
			line = line[:60] + "…"
		}
		return line
	}
	return ""
}

// detectedAdapterKeys builds a set of adapter types from the detected list.
func detectedAdapterKeys(detected []detectedVPN) map[string]bool {
	keys := make(map[string]bool)
	for _, d := range detected {
		if d.adapterType != "" {
			keys[d.adapterType] = true
		}
	}
	return keys
}

// detectedBinaryPath returns the binary detected for an adapter type, or "".
func detectedBinaryPath(detected []detectedVPN, adapterType string) string {
	for _, d := range detected {
		if d.adapterType == adapterType && d.path != "" {
			return d.path
		}
	}
	return ""
}

// binaryPathPlan decides how the wizard handles binary_path for one profile:
// whether to ask at all, and the value to prefill. The detected path is kept
// as the suggestion; when the adapter already finds that binary on its own
// (PATH or its standard install location) nothing is asked and nothing is
// written, to keep the YAML free of redundant overrides.
func binaryPathPlan(adapterType, detectedPath string, isStandard func(adapterType, path string) bool) (ask bool, suggestion string) {
	if adapterType == "cloudflarewarp" {
		return false, ""
	}
	if detectedPath != "" && isStandard(adapterType, detectedPath) {
		return false, ""
	}
	return true, detectedPath
}
