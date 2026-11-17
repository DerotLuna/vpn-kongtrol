// Package vpndetect locates installed VPN client binaries. It is shared by
// the setup wizard (which scans broadly so it can show what is installed)
// and the connect preflight / doctor (which need to know whether the binary
// an adapter will actually execute is present). OS-specific knowledge lives
// in build-tagged files (probes_windows.go, probes_darwin.go, probes_other.go).
package vpndetect

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
)

// Probe describes how the wizard locates one VPN client on this OS.
type Probe struct {
	Label       string
	AdapterType string   // "" when Kongtrol has no adapter for the client
	PathNames   []string // names resolved through PATH (exec.LookPath)
	SearchDirs  []string // directories walked looking for ExeNames
	SearchDepth int
	ExeNames    []string // file names matched case-insensitively
	VersionArgs []string // args that print a version; nil = do not execute
}

// Client is one detected VPN client.
type Client struct {
	Label       string
	AdapterType string
	Path        string
	VersionArgs []string
}

// standard is where an adapter itself looks for its binary.
type standard struct {
	locations   []string // absolute paths or PATH names, in adapter order
	versionArgs []string // nil = do not execute (GUI app); read file version instead
}

// Probes returns a copy of the wizard probes for this OS.
func Probes() []Probe {
	out := make([]Probe, len(probes))
	copy(out, probes)
	return out
}

// DetectAll runs every probe and returns the clients found.
func DetectAll() []Client {
	var found []Client
	for _, p := range probes {
		if path, ok := Detect(p); ok {
			found = append(found, Client{Label: p.Label, AdapterType: p.AdapterType, Path: path, VersionArgs: p.VersionArgs})
		}
	}
	return found
}

// Detect resolves one probe. When the adapter's own standard location has the
// binary, that path wins so callers can tell it needs no binary_path override.
func Detect(p Probe) (string, bool) {
	if p.AdapterType != "" {
		if path, ok := FindStandard(p.AdapterType); ok {
			return path, true
		}
	}
	for _, name := range p.PathNames {
		if path, err := exec.LookPath(name); err == nil {
			return path, true
		}
	}
	var candidates []string
	for _, dir := range p.SearchDirs {
		walkExe(dir, p.ExeNames, p.SearchDepth, &candidates)
	}
	if len(candidates) == 0 {
		return "", false
	}
	sort.Strings(candidates)
	return candidates[len(candidates)-1], true
}

// StandardLocations lists where the adapter of this type looks for its
// binary on this OS. Empty means unknown or the adapter is unsupported here.
// OpenVPN is resolved by openvpn.FindBinary and is not listed.
func StandardLocations(adapterType string) []string {
	return append([]string(nil), standards[adapterType].locations...)
}

// VersionArgs returns the args that print the adapter binary's version, or
// nil when the binary must not be executed (GUI app).
func VersionArgs(adapterType string) []string {
	if adapterType == "openvpn" {
		return []string{"--version"}
	}
	return append([]string(nil), standards[adapterType].versionArgs...)
}

// FindStandard returns the binary the adapter of this type would execute
// without any binary_path override.
func FindStandard(adapterType string) (string, bool) {
	if adapterType == "openvpn" {
		p, err := openvpn.FindBinary("")
		return p, err == nil
	}
	for _, loc := range standards[adapterType].locations {
		if filepath.IsAbs(loc) {
			if fi, err := os.Stat(loc); err == nil && fi.Mode().IsRegular() {
				return loc, true
			}
			continue
		}
		if p, err := exec.LookPath(loc); err == nil {
			return p, true
		}
	}
	return "", false
}

// Known reports whether this OS has standard locations for the adapter type.
func Known(adapterType string) bool {
	return adapterType == "openvpn" || len(standards[adapterType].locations) > 0
}

// IsStandard reports whether path is the binary the adapter already finds on
// its own, i.e. writing it as binary_path would only add YAML noise.
func IsStandard(adapterType, path string) bool {
	if path == "" {
		return false
	}
	found, ok := FindStandard(adapterType)
	return ok && samePath(found, path)
}

func walkExe(dir string, names []string, depth int, out *[]string) {
	if depth < 0 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			walkExe(full, names, depth-1, out)
			continue
		}
		for _, want := range names {
			if strings.EqualFold(e.Name(), want) {
				*out = append(*out, full)
				break
			}
		}
	}
}
