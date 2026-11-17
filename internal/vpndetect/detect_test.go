package vpndetect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectWalksSearchDirsCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "Vendor", "v2.1")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(sub, "MyClient.EXE")
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := Probe{Label: "x", SearchDirs: []string{dir}, SearchDepth: 2, ExeNames: []string{"myclient.exe"}}
	got, ok := Detect(p)
	if !ok || got != exe {
		t.Fatalf("Detect = %q, %v; want %q", got, ok, exe)
	}
	p.SearchDepth = 0
	if _, ok := Detect(p); ok {
		t.Fatal("depth 0 must not descend into subdirectories")
	}
}

func TestDetectMissing(t *testing.T) {
	p := Probe{Label: "x", PathNames: []string{"kongtrol-no-such-binary-xyz"}, SearchDirs: []string{t.TempDir()}, ExeNames: []string{"nope"}}
	if _, ok := Detect(p); ok {
		t.Fatal("expected not found")
	}
}

func TestEveryProbeTypeIsKnownOrUnsupported(t *testing.T) {
	for _, p := range Probes() {
		if p.AdapterType == "" {
			continue
		}
		// Every adapter probe must either have standard locations on this OS
		// or be a type the adapter deliberately does not support here.
		if !Known(p.AdapterType) && p.AdapterType != "protonvpn" && p.AdapterType != "nordvpn" && p.AdapterType != "globalprotect" {
			t.Errorf("probe %q: adapter type %q has no standard locations", p.Label, p.AdapterType)
		}
	}
}

func TestIsStandardEmptyPath(t *testing.T) {
	if IsStandard("tailscale", "") {
		t.Fatal("empty path must never be standard")
	}
	if IsStandard("no-such-adapter", "/x") {
		t.Fatal("unknown adapter must never be standard")
	}
}
