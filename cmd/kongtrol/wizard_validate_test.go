package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestValidateHost(t *testing.T) {
	fn := validateHost(i18n.EN)
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"plain hostname", "vpn.example.com", false},
		{"single label host", "vpn-office", false},
		{"ipv4", "192.168.1.1", false},
		{"ipv6", "::1", false},
		{"has scheme", "https://vpn.example.com", true},
		{"has space", "vpn example.com", true},
		{"empty label", "vpn..example.com", true},
		{"trailing dot label", "vpn.example.com.", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := fn(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateHost(%q) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	fn := validatePort(i18n.EN)
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"empty is allowed", "", false},
		{"valid low", "1", false},
		{"valid high", "65535", false},
		{"valid typical", "443", false},
		{"zero", "0", true},
		{"too high", "65536", true},
		{"negative", "-1", true},
		{"not a number", "abc", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := fn(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validatePort(%q) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateExistingPath(t *testing.T) {
	fn := validateExistingPath(i18n.EN)

	dir := t.TempDir()
	existing := filepath.Join(dir, "config.ovpn")
	if err := os.WriteFile(existing, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := fn(""); err != nil {
		t.Errorf("empty path should be allowed, got %v", err)
	}
	if err := fn(existing); err != nil {
		t.Errorf("existing path should be allowed, got %v", err)
	}
	if err := fn(filepath.Join(dir, "missing.ovpn")); err == nil {
		t.Error("missing path should return an error")
	}
}

func TestValidateProfileName(t *testing.T) {
	fn := validateProfileName(i18n.EN)

	if err := fn(""); err == nil {
		t.Error("empty name should return an error")
	}
	if err := fn("   "); err == nil {
		t.Error("whitespace-only name should return an error")
	}
	if err := fn("office-vpn"); err != nil {
		t.Errorf("valid name should be allowed, got %v", err)
	}
}
