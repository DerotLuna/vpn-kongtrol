package api

import (
	"net/http"

	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

// DetectedClient is one VPN client installed on this computer
// (GET /api/v1/vpns/detect). AdapterType is "" when Kongtrol has no adapter
// for it, so the dashboard can still say "found, not supported".
type DetectedClient struct {
	Label       string `json:"label"`
	AdapterType string `json:"adapter_type"`
	Path        string `json:"path"`
}

// detectClients is replaced in tests.
var detectClients = vpndetect.DetectAll

// GET /api/v1/vpns/detect — the VPN clients the setup wizard (`kongtrol
// init`) would find, for the dashboard's "create your first VPN" guide.
// Read-only and local: it only looks for files and PATH entries (the same
// internal/vpndetect probes as the wizard, doctor and connect preflight),
// never runs a client and never contacts anything.
func (s *Server) handleDetectVPNs(w http.ResponseWriter, _ *http.Request) {
	found := detectClients()
	out := make([]DetectedClient, 0, len(found))
	for _, c := range found {
		out = append(out, DetectedClient{Label: c.Label, AdapterType: c.AdapterType, Path: c.Path})
	}
	writeJSON(w, http.StatusOK, out)
}
