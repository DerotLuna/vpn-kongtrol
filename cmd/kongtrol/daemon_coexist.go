package main

import (
	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

// providerAppLabels are the product names shown when a provider app's own
// tunnel blocks a connect (brand names: not translated).
var providerAppLabels = map[string]string{
	"protonvpn": "Proton VPN",
	"nordvpn":   "NordVPN",
	"surfshark": "Surfshark",
}

// activeProviderTunnel is appattach.ActiveTunnel (replaced in tests).
var activeProviderTunnel = appattach.ActiveTunnel

// externalFullTunnels lists the consumer VPN apps whose tunnel is up now,
// found by their interface (no admin, nothing launched). An interface
// without an address is ignored: a retained adapter is not a session
// (docs/research/2026-11-21-vpn-coexistence.md, ADR-0099).
func externalFullTunnels() []app.ExternalTunnel {
	var out []app.ExternalTunnel
	for _, typ := range app.FullTunnelAppTypes() {
		t, up, known := activeProviderTunnel(typ)
		if !known || !up || t.IP == nil {
			continue
		}
		label := providerAppLabels[typ]
		if label == "" {
			label = typ
		}
		out = append(out, app.ExternalTunnel{Type: typ, Label: label, Interface: t.Name})
	}
	return out
}

// onFullTunnelSuperseded tells every client, once, that a dropped full
// tunnel is not blocked because another VPN carries all traffic now.
func onFullTunnelSuperseded(profile, by string) {
	watchdogNotice("WARN", "killswitch.superseded", "killswitch.event.superseded", profile, by)
}

// checkConnectConflict is the synchronous check of POST .../connect.
func checkConnectConflict(name string) error {
	if svc := profileSvc.Load(); svc != nil {
		return svc.CheckFullTunnelConflict(name)
	}
	return nil
}
