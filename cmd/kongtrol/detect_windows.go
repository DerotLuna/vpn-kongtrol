//go:build windows

package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// peVersion reads the ProductVersion from a Windows PE binary's version resource.
// Returns an empty string if the version cannot be determined.
func peVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return fileVersion(ctx, path)
}

// fileVersion reads the PE ProductVersion without launching the binary, so
// GUI clients (FortiClient.exe, wireguard.exe) are never opened.
func fileVersion(ctx context.Context, path string) string {
	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf("(Get-Item -LiteralPath '%s').VersionInfo.ProductVersion", strings.ReplaceAll(path, "'", "''"))).Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	ver := strings.TrimSpace(string(out))
	if ver == "" || strings.Contains(ver, "Exception") {
		return ""
	}
	return ver
}
