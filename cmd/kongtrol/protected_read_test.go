package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A root-only file is imported from what the elevated read returns, after
// the same checks; a dismissed prompt or no way to ask keeps the "what to
// do" line (#5 #7, ADR-0058).
func TestImportRootOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	orig := readProtectedFn
	t.Cleanup(func() { readProtectedFn = orig })
	src := "/etc/wireguard/wg0.conf"

	readProtectedFn = func(_ context.Context, path string) ([]byte, error) {
		if path != src {
			t.Fatalf("read %q", path)
		}
		return []byte("[Interface]\nPrivateKey = a\nAddress = 10.0.0.2/32\n"), nil
	}
	res, err := importRootOnly(src, "", false, true)
	if err != nil || res.Path != filepath.Join(home, ".kongtrol", "configs", "wg0.conf") || res.Type != "wireguard" {
		t.Fatalf("import: %+v %v", res, err)
	}

	for name, readErr := range map[string]error{"cancelled": errElevatedReadCancelled, "unavailable": errElevatedReadUnavailable} {
		readProtectedFn = func(context.Context, string) ([]byte, error) { return nil, readErr }
		_, err := importRootOnly(src, "other", false, true)
		if err == nil || !strings.Contains(err.Error(), "kongtrol profile import "+src) {
			t.Fatalf("%s: want the what-to-do line, got %v", name, err)
		}
	}

	readProtectedFn = func(context.Context, string) ([]byte, error) {
		return []byte("[Interface]\nPrivateKey = a\nPostUp = id\n"), nil
	}
	if _, err := importRootOnly(src, "hook", false, true); err == nil {
		t.Fatal("a hook script passed after the elevated read")
	}
}

// The elevated command runs as root: its refusal (stderr, shown to the
// user by sudo/pkexec's caller) is the generic one and never names a
// section, key or line of the file (ADR-0063).
func TestProfileReadProtectedCmd_GenericRefusal(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"hook.conf":  "[Interface]\nPrivateKey = a\nPostUp = id\n",
		"key.conf":   "[Interface]\nPrivateKey = a\nRootSecret = x\n",
		"sect.conf":  "[Interface]\nPrivateKey = a\n[RootSection]\n",
		"line.ovpn":  "client\nremote vpn.example.com\n</RootTag>\n",
		"plain.ovpn": "RootLine one two\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		err := profileReadProtectedCmd.RunE(profileReadProtectedCmd, []string{p})
		if err == nil {
			t.Fatalf("%s: released", name)
		}
		for _, leak := range []string{"PostUp", "Root"} {
			if strings.Contains(err.Error(), leak) {
				t.Fatalf("%s: refusal echoes the file: %v", name, err)
			}
		}
		if !strings.Contains(err.Error(), filepath.Base(p)) {
			t.Fatalf("%s: want the generic refusal naming the path, got %v", name, err)
		}
	}
}
