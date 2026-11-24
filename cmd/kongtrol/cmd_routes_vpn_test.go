package main

import (
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// `kongtrol routes vpn` names the origin of each route, folds a full tunnel
// into "all traffic" in words, says who wins a conflict and never leaves a
// key or a broken format verb behind (ADR-0107).
func TestRenderVPNOwnRoutes(t *testing.T) {
	list := []app.ProfileOwnRoutes{
		{Name: "work", Type: "openvpn", Connected: true, Source: app.OwnSourceConfig, AllTraffic: true, Routes: []app.OwnRoute{
			{Destination: "0.0.0.0/1", Origin: app.OwnRouteServer},
			{Destination: "128.0.0.0/1", Origin: app.OwnRouteServer},
			{Destination: "10.0.0.0/8", Origin: app.OwnRouteFile, Conflicts: []app.RouteConflict{
				{Rule: "db", Via: "other", Destination: "10.1.2.3/32", Winner: app.WinnerRule},
				{Rule: "lan", Via: "other", Destination: "10.0.0.0/8", Winner: app.WinnerSystem},
			}},
			{Destination: "172.16.5.0/24", Origin: app.OwnRouteFile, Conflicts: []app.RouteConflict{
				{Rule: "wide", Via: "other", Destination: "172.16.0.0/16", Winner: app.WinnerVPN},
			}},
		}},
		{Name: "idle", Type: "openvpn", Source: app.OwnSourceConfig, ServerPending: true, Routes: []app.OwnRoute{}},
		{Name: "home", Type: "wireguard", Source: app.OwnSourceConfig, RulesOnly: true, Routes: []app.OwnRoute{{Destination: "0.0.0.0/0", Origin: app.OwnRouteFile}}},
		{Name: "proton", Type: "protonvpn", Source: app.OwnSourceApp, AllTraffic: true, Routes: []app.OwnRoute{}},
	}
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		t.Setenv("KONGTROL_LANG", string(lang))
		text := strings.Join(renderVPNOwnRoutes(list), "\n")
		for _, want := range []string{
			i18n.T(lang, "cli.routes.vpn.all_traffic"),
			i18n.T(lang, "cli.routes.vpn.origin.file"),
			i18n.F(lang, "cli.routes.vpn.conflict.rule", "db", "10.1.2.3/32", "other"),
			i18n.F(lang, "cli.routes.vpn.conflict.vpn", "172.16.5.0/24", "wide"),
			i18n.F(lang, "cli.routes.vpn.conflict.system", "lan", "10.0.0.0/8", "other"),
			i18n.T(lang, "cli.routes.vpn.server_pending"),
			i18n.T(lang, "cli.routes.vpn.rules_only"),
			i18n.T(lang, "cli.routes.vpn.app_all"),
			i18n.T(lang, "cli.routes.vpn.readonly"),
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: output lacks %q\n%s", lang, want, text)
			}
		}
		if strings.Contains(text, "0.0.0.0/1") || strings.Contains(text, "cli.routes.") || strings.Contains(text, "%!") {
			t.Errorf("%s: raw half route, key or format error in output\n%s", lang, text)
		}
	}
}
