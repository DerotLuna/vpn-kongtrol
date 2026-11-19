package api

import (
	"net/http"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

// DetectedClient is one VPN client installed on this computer
// (GET /api/v1/vpns/detect). AdapterType is "" when Kongtrol has no adapter
// for it, so the dashboard can still say "found, not supported".
//
// Active says whether the client's tunnel is up right now: true/false for
// provider apps Kongtrol attaches to (their tunnel interface is known),
// null when Kongtrol cannot tell for that client (ADR-0053). Interface is
// the tunnel interface when Active is true.
type DetectedClient struct {
	Label       string `json:"label"`
	AdapterType string `json:"adapter_type"`
	Path        string `json:"path"`
	Active      *bool  `json:"active"`
	Interface   string `json:"interface,omitempty"`
}

// detectClients and activeTunnel are replaced in tests.
var (
	detectClients = vpndetect.DetectAll
	activeTunnel  = appattach.ActiveTunnel
)

// GET /api/v1/vpns/detect — the VPN clients the setup wizard (`kongtrol
// init`) would find, for the dashboard's "create your first VPN" guide.
// Read-only and local: it only looks for files and PATH entries (the same
// internal/vpndetect probes as the wizard, doctor and connect preflight)
// and lists network interfaces (plus `scutil --nc list` on macOS) to see
// whether a provider app's tunnel is up. It never runs or automates a
// client, needs no admin rights and never contacts anything.
func (s *Server) handleDetectVPNs(w http.ResponseWriter, _ *http.Request) {
	found := detectClients()
	out := make([]DetectedClient, 0, len(found))
	for _, c := range found {
		d := DetectedClient{Label: c.Label, AdapterType: c.AdapterType, Path: c.Path}
		if c.AdapterType != "" {
			if t, up, known := activeTunnel(c.AdapterType); known {
				d.Active = &up
				if up {
					d.Interface = t.Name
				}
			}
		}
		out = append(out, d)
	}
	writeJSON(w, http.StatusOK, out)
}
