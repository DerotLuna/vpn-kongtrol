package main

import (
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// The native menu shows only the offers it knows; "Agregar <App>" opens the
// sheet on a provider from the closed list only (ADR-0102).
func TestOfferOf(t *testing.T) {
	cases := []struct {
		e    monitor.Event
		ok   bool
		want trayOffer
	}{
		{monitor.Event{Offer: offerEnableKillSwitch, Profile: "work"}, true, trayOffer{kind: offerEnableKillSwitch, profile: "work"}},
		{monitor.Event{Offer: offerReleaseBlock, Profile: "work"}, true, trayOffer{kind: offerReleaseBlock, profile: "work"}},
		{monitor.Event{Offer: offerReleaseBlock}, false, trayOffer{}},
		{monitor.Event{Offer: offerAddProvider, Profile: "work", Args: []string{"work", "protonvpn"}}, true,
			trayOffer{kind: offerAddProvider, profile: "work", provider: "protonvpn"}},
		{monitor.Event{Offer: offerAddProvider, Args: []string{"work", "../evil"}}, false, trayOffer{}},
		{monitor.Event{Offer: offerAddProvider}, false, trayOffer{}},
		{monitor.Event{Offer: "something_new"}, false, trayOffer{}},
		{monitor.Event{}, false, trayOffer{}},
	}
	for _, c := range cases {
		got, ok := offerOf(c.e)
		if ok != c.ok || got != c.want {
			t.Errorf("offerOf(%+v) = %+v, %v; want %+v, %v", c.e, got, ok, c.want, c.ok)
		}
	}
}

func TestAddProviderPath(t *testing.T) {
	if got := addProviderPath("protonvpn"); got != "/?add=vpn&type=protonvpn" {
		t.Errorf("protonvpn: %q", got)
	}
	for _, bad := range []string{"", "wireguard", "protonvpn&add=rule", "../x"} {
		if got := addProviderPath(bad); got != "" {
			t.Errorf("addProviderPath(%q) = %q, want none", bad, got)
		}
	}
}

func TestOfferTitle(t *testing.T) {
	if got := (trayOffer{kind: offerAddProvider, provider: "protonvpn"}).title(); got != trayF("offer.action.add_provider", "Proton") {
		t.Errorf("title = %q", got)
	}
	if got := (trayOffer{kind: offerEnableKillSwitch}).title(); got == "offer.action.enable_killswitch" {
		t.Error("enable_killswitch title not translated")
	}
}
