package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

// routesVPNCmd is `kongtrol routes vpn [perfil]`: the routes each VPN brings
// by itself (.ovpn, server, WireGuard AllowedIPs), read-only, with the
// rules that overlap them — the panel's "Rutas que trae cada VPN" (ADR-0107).
var routesVPNCmd = &cobra.Command{
	Use:  "vpn [perfil]",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		list, err := vpnOwnRoutes(name)
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				Profiles []app.ProfileOwnRoutes `json:"profiles"`
			}{list})
		}
		if outputQuiet {
			return nil
		}
		for _, line := range renderVPNOwnRoutes(list) {
			fmt.Fprintln(cmd.OutOrStdout(), line)
		}
		return nil
	},
}

func init() {
	routesCmd.AddCommand(routesVPNCmd)
}

// vpnOwnRoutes asks the running daemon (it knows what the servers pushed);
// with no daemon it reads the profile files itself (nothing is connected).
func vpnOwnRoutes(name string) ([]app.ProfileOwnRoutes, error) {
	if base := daemonAPIBase(); cfgPath == "" && daemonProbeFn(base) {
		path := base + "/api/v1/vpns/routes"
		if name != "" {
			path += "?profile=" + url.QueryEscape(name)
		}
		req, err := daemonRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		resp, err := daemonctl.HTTPClient(15 * time.Second).Do(req)
		if err != nil {
			return nil, fmt.Errorf("%s", cf("cli.routes.error.list", err))
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%s", cf("cli.routes.unknown", name))
		}
		if resp.StatusCode != http.StatusOK {
			return nil, daemonAPIError(resp)
		}
		var out struct {
			Profiles []app.ProfileOwnRoutes `json:"profiles"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		if out.Profiles == nil {
			out.Profiles = []app.ProfileOwnRoutes{}
		}
		return out.Profiles, nil
	}
	cfg, err := config.Load(activeCfgPath)
	if err != nil {
		return nil, localizeConfigErr(err)
	}
	if _, ok := cfg.VPNs[name]; name != "" && !ok {
		return nil, fmt.Errorf("%s", cf("cli.routes.unknown", name))
	}
	return app.OwnRoutes(cfg, nil, name), nil
}

// renderVPNOwnRoutes is the terminal view: per profile a summary line,
// then each route with its origin and the rules that overlap it.
func renderVPNOwnRoutes(list []app.ProfileOwnRoutes) []string {
	var out []string
	if len(list) == 0 {
		return []string{tuiInfo(paint(cDim, ct("cli.routes.vpn.none")))}
	}
	for _, p := range list {
		state := ct("cli.routes.vpn.disconnected")
		if p.Connected {
			state = ct("cli.routes.vpn.connected")
		}
		out = append(out, "", "  "+paintBold(cBright, p.Name)+paint(cDim, " · "+p.Type+" · "+state))
		switch p.Source {
		case app.OwnSourceApp:
			if p.AllTraffic {
				out = append(out, "    "+ct("cli.routes.vpn.app_all"))
			} else {
				out = append(out, "    "+ct("cli.routes.vpn.app"))
			}
			continue
		case app.OwnSourceUnreadable:
			out = append(out, "    "+paint(cDim, ct("cli.routes.vpn.unreadable")))
		}
		full := app.CarriesAllTraffic(p.Routes)
		if full {
			origin := ""
			for _, r := range p.Routes {
				if isFullTunnelPrefix(r.Destination) {
					origin = r.Origin
					break
				}
			}
			out = append(out, fmt.Sprintf("    %-24s %s", ct("cli.routes.vpn.all_traffic"), paint(cDim, ct("cli.routes.vpn.origin."+origin))))
		} else if p.AllTraffic {
			out = append(out, "    "+ct("cli.routes.vpn.all_traffic"))
		}
		for _, r := range p.Routes {
			if full && isFullTunnelPrefix(r.Destination) {
				continue
			}
			out = append(out, fmt.Sprintf("    %-24s %s", r.Destination, paint(cDim, ct("cli.routes.vpn.origin."+r.Origin))))
			for _, c := range r.Conflicts {
				out = append(out, "      "+conflictLine(r.Destination, c))
			}
		}
		if len(p.Routes) == 0 && p.Source == app.OwnSourceConfig {
			out = append(out, "    "+paint(cDim, ct("cli.routes.vpn.no_file_routes")))
		}
		if p.RulesOnly {
			out = append(out, "    "+paint(cDim, ct("cli.routes.vpn.rules_only")))
		}
		if p.ServerPending {
			out = append(out, "    "+paint(cDim, ct("cli.routes.vpn.server_pending")))
		}
		if p.AllTraffic && !p.RulesOnly {
			out = append(out, "    "+paint(cDim, ct("cli.routes.vpn.all_wins")))
		}
	}
	out = append(out, "", "  "+ct("cli.routes.vpn.readonly"), "  "+paint(cDim, ct("cli.routes.vpn.precedence")),
		"  "+paint(cDim, ct("cli.routes.vpn.next")))
	return out
}

// isFullTunnelPrefix: 0.0.0.0/0 or one of its /1 halves, shown as "all
// traffic" in words.
func isFullTunnelPrefix(cidr string) bool {
	return cidr == "0.0.0.0/0" || cidr == "0.0.0.0/1" || cidr == "128.0.0.0/1"
}

// conflictLine says who wins between a rule and a VPN route (ADR-0107).
func conflictLine(route string, c app.RouteConflict) string {
	switch c.Winner {
	case app.WinnerRule:
		return cf("cli.routes.vpn.conflict.rule", c.Rule, c.Destination, c.Via)
	case app.WinnerVPN:
		return cf("cli.routes.vpn.conflict.vpn", route, c.Rule)
	}
	return cf("cli.routes.vpn.conflict.system", c.Rule, c.Destination, c.Via)
}
