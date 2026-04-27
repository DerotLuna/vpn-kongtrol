//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// peVersion reads the ProductVersion from a Windows PE binary's version resource.
// Returns an empty string if the version cannot be determined.
func peVersion(path string) string {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf("(Get-Item '%s').VersionInfo.ProductVersion", path)).Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	ver := strings.TrimSpace(string(out))
	if ver == "" || strings.Contains(ver, "Exception") {
		return ""
	}
	return ver
}
