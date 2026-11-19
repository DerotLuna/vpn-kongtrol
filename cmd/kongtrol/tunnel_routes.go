package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

// apiTunnelRoute / apiTunnelRoutes / apiTunnelRoutesResponse mirror
// GET /api/v1/tunnels/routes (internal/api/handlers_tunnel_routes.go).
type apiTunnelRoute struct {
	Destination string `json:"destination"`
	Origin      string `json:"origin"`
	Policy      string `json:"policy,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
}

type apiTunnelRoutes struct {
	Name       string           `json:"name"`
	Connected  bool             `json:"connected"`
	Interface  string           `json:"interface,omitempty"`
	AllTraffic bool             `json:"all_traffic"`
	Total      int              `json:"total"`
	Counts     map[string]int   `json:"counts"`
	Routes     []apiTunnelRoute `json:"routes"`
}

type apiTunnelRoutesResponse struct {
	RestVia       string            `json:"rest_via,omitempty"`
	RestViaTunnel string            `json:"rest_via_tunnel,omitempty"`
	Tunnels       []apiTunnelRoutes `json:"tunnels"`
}

// errTunnelNotFound: the daemon does not know the profile.
var errTunnelNotFound = fmt.Errorf("tunnel not found")

// fetchTunnelRoutes asks the daemon what goes through each connected
// tunnel (name == "") or through one profile.
func fetchTunnelRoutes(base, name string, timeout time.Duration) (apiTunnelRoutesResponse, error) {
	var out apiTunnelRoutesResponse
	path := "/api/v1/tunnels/routes"
	if name != "" {
		path = "/api/v1/tunnels/" + url.PathEscape(name) + "/routes"
	}
	req, err := daemonRequest(http.MethodGet, base+path, nil)
	if err != nil {
		return out, err
	}
	resp, err := daemonctl.HTTPClient(timeout).Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return out, errTunnelNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return out, daemonAPIError(resp)
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

// runTunnelRoutes is `kongtrol routes [perfil]`: per connected profile,
// what goes through its tunnel grouped by origin, then where everything
// else goes.
func runTunnelRoutes(name string) error {
	base := daemonAPIBase()
	if !probeDaemonAPI(base) {
		if outputJSON {
			return emitJSON(apiTunnelRoutesResponse{Tunnels: []apiTunnelRoutes{}})
		}
		fmt.Println(tuiInfo(paint(cDim, ct("cli.reload.error.no_daemon"))))
		return nil
	}
	res, err := fetchTunnelRoutes(base, name, 15*time.Second)
	if err == errTunnelNotFound {
		return fmt.Errorf("%s", cf("cli.routes.unknown", name))
	}
	if err != nil {
		return fmt.Errorf("%s", cf("cli.routes.error.list", err))
	}
	if outputJSON {
		return emitJSON(res)
	}
	connected := 0
	for _, t := range res.Tunnels {
		if !t.Connected {
			fmt.Println(tuiInfo(paint(cDim, cf("cli.routes.not_connected", t.Name, t.Name))))
			continue
		}
		connected++
		fmt.Println()
		for _, line := range renderTunnelRoutes(t, terminalWidth()) {
			fmt.Println(line)
		}
	}
	if connected == 0 {
		if name == "" {
			fmt.Println(tuiInfo(paint(cDim, ct("cli.routes.no_tunnels"))))
		}
		return nil
	}
	fmt.Println()
	fmt.Println("  " + restLine(res))
	return nil
}

// restLine is the one-line summary of where everything else goes.
func restLine(res apiTunnelRoutesResponse) string {
	switch {
	case res.RestViaTunnel != "":
		return cf("cli.routes.rest_tunnel", res.RestViaTunnel)
	case res.RestVia != "":
		return cf("cli.routes.rest", res.RestVia)
	}
	return ct("cli.routes.rest_unknown")
}

// routeGroup is one origin block of a tunnel's routes.
type routeGroup struct {
	origin, policy string
	dests          []string
}

// groupTunnelRoutes keeps the API order (server, policy by name, vpn) and
// splits it into blocks.
func groupTunnelRoutes(routes []apiTunnelRoute) []routeGroup {
	var groups []routeGroup
	for _, r := range routes {
		n := len(groups)
		if n == 0 || groups[n-1].origin != r.Origin || groups[n-1].policy != r.Policy {
			groups = append(groups, routeGroup{origin: r.Origin, policy: r.Policy})
			n++
		}
		groups[n-1].dests = append(groups[n-1].dests, r.Destination)
	}
	return groups
}

func renderTunnelRoutes(t apiTunnelRoutes, width int) []string {
	head := paintBold(cBright, t.Name) + "  " + paint(cDim, cf("cli.routes.tunnel.header", t.Total))
	if t.Interface != "" {
		head += paint(cDim, " · "+t.Interface)
	}
	lines := []string{"  " + head}
	if t.AllTraffic {
		lines = append(lines, "    "+tuiWarn(ct("routes.all_traffic")))
	}
	if t.Total == 0 && !t.AllTraffic {
		lines = append(lines, "    "+paint(cDim, ct("cli.routes.tunnel.none")))
	}
	for _, g := range groupTunnelRoutes(t.Routes) {
		var title string
		switch g.origin {
		case "server":
			title = cf("cli.routes.group.server", len(g.dests))
		case "policy":
			title = cf("cli.routes.group.policy", g.policy, len(g.dests))
		default:
			title = cf("cli.routes.group.vpn", len(g.dests))
		}
		lines = append(lines, "    "+styleMapHdr.Render(title))
		if g.origin == "server" {
			lines = append(lines, "    "+paint(cDim, ct("routes.server_explain")))
		}
		lines = append(lines, packColumns(g.dests, width, 6)...)
	}
	return lines
}

// packColumns lays dests out in as many columns as fit width.
func packColumns(items []string, width, indent int) []string {
	cell := 0
	for _, s := range items {
		cell = max(cell, lipgloss.Width(s)+2)
	}
	cols := 1
	if cell > 0 {
		cols = max(1, (width-indent)/cell)
	}
	var lines []string
	pad := strings.Repeat(" ", indent)
	for i := 0; i < len(items); i += cols {
		var b strings.Builder
		for j := i; j < min(i+cols, len(items)); j++ {
			if j > i {
				b.WriteString(strings.Repeat(" ", cell-lipgloss.Width(items[j-1])))
			}
			b.WriteString(styleMapName.Render(items[j]))
		}
		lines = append(lines, pad+b.String())
	}
	return lines
}

// tunnelRoutesHint is the short per-tunnel note under `status` and after
// `up`: "N rutas del servidor" or "todo tu tráfico". Empty when there is
// nothing to say.
func tunnelRoutesHint(t apiTunnelRoutes) string {
	switch {
	case t.AllTraffic:
		return ct("cli.status.routes.all")
	case t.Counts["server"] > 0:
		return cf("cli.status.routes.server", t.Counts["server"])
	case t.Total > 0:
		return cf("cli.status.routes.tunnel", t.Total)
	}
	return ""
}

// renderStatusRoutesLines asks the daemon at base for each connected
// tunnel's routes and returns one short line per tunnel. Silent on error.
func renderStatusRoutesLines(base string) []string {
	res, err := fetchTunnelRoutes(base, "", 4*time.Second)
	if err != nil {
		return nil
	}
	var lines []string
	for _, t := range res.Tunnels {
		hint := tunnelRoutesHint(t)
		if !t.Connected || hint == "" {
			continue
		}
		lines = append(lines, "  "+tuiInfo(paintBold(cBright, t.Name)+"  "+paint(cDim, hint+" · "+cf("cli.status.routes.see", t.Name))))
	}
	return lines
}

// upRoutesLine is printed once after a profile connects in `up`:
// "ekiipago-ovpn: 25 rutas enviadas por el servidor · ver: kongtrol routes ekiipago-ovpn".
func upRoutesLine(base, name string) string {
	res, err := fetchTunnelRoutes(base, name, 4*time.Second)
	if err != nil || len(res.Tunnels) == 0 {
		return ""
	}
	t := res.Tunnels[0]
	switch {
	case t.AllTraffic:
		return cf("cli.up.routes.all", name, name)
	case t.Counts["server"] > 0:
		return cf("cli.up.routes.server", name, t.Counts["server"], name)
	}
	return ""
}
