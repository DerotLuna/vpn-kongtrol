package main

import (
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// newPublicIPChecker builds the opt-in public IP lookup from cfg.
func newPublicIPChecker(c *config.Config) *security.LeakTester {
	return security.NewLeakTester(security.LeakTesterOptions{
		Enabled:     c.PublicIPCheckEnabled(),
		Endpoints:   c.PublicIPEndpoints(),
		ExpectedIPs: c.Security.IntegrityCheck.ExpectedIPs,
	})
}

// applyPublicIPConfig hot-applies a config change to the running checker.
func applyPublicIPConfig(c *config.Config) {
	if leak == nil || c == nil {
		return
	}
	wasOn := leak.Enabled()
	leak.Configure(c.PublicIPCheckEnabled(), c.PublicIPEndpoints(), c.Security.IntegrityCheck.ExpectedIPs)
	if !wasOn && leak.Enabled() {
		leak.Trigger() // just turned on by the user: show the IP soon
	}
}

// triggerPublicIPCheck asks for a debounced lookup after a tunnel event; a
// no-op unless the user opted in.
func triggerPublicIPCheck() {
	if leak != nil {
		leak.Trigger()
	}
}
