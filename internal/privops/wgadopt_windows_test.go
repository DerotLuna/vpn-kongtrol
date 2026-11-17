//go:build windows

package privops

import "testing"

// Windows tunnel service names may exceed Linux's 15-char IFNAMSIZ; this is the
// real case that motivated adoption (a tunnel created by an older Kongtrol).
func TestWGSetAllowedIPs_AdoptsLongWindowsName(t *testing.T) {
	testWGSetAllowedIPsAdopts(t, "us-content-US-FREE-1")
}
