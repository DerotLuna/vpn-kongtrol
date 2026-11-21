package api

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configexport"
	"github.com/vpn-kongtrol/kongtrol/internal/doctor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// runDoctor is doctor.Run; tests replace it.
var runDoctor = doctor.Run

// GET /api/v1/doctor — the same checks as `kongtrol doctor`, run inside
// the daemon with its own tunnel and security state, translated to the
// request's language. A check's "fix" is only kept when this daemon can
// run it (POST /api/v1/helper/install).
func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	sec := s.securityState()
	dm := &doctor.Daemon{
		Tunnels:           map[string]vpn.Status{},
		TunnelInterfaces:  map[string]string{},
		KillSwitchEnabled: sec.killEnabled,
		KillSwitchActive:  sec.killActive,
		KillSwitch:        sec.ksStatus,
		DNSGuardActive:    sec.dnsActive,
	}
	if s.collector != nil {
		for name, m := range s.collector.Snapshot() {
			dm.Tunnels[name] = m.Status
			dm.TunnelInterfaces[name] = m.InterfaceName
		}
	}
	rep := runDoctor(doctor.Options{
		Lang:       requestLang(r),
		ConfigPath: s.configPath,
		Version:    s.version,
		Daemon:     dm,
	})
	if s.helperInstall == nil {
		for i := range rep.Checks {
			rep.Checks[i].Fix = ""
		}
	}
	writeJSON(w, http.StatusOK, rep)
}

// GET /api/v1/config/export — kongtrol.yaml as a downloadable template
// (no secrets: passwords live in the OS keychain). 404 with a translated
// hint while there is no config yet.
func (s *Server) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if errors.Is(err, errSetupNoConfig) || (err == nil && len(cfg.VPNs) == 0) {
		writeLocalizedError(w, r, http.StatusNotFound, &config.ValidationError{Key: "export.err.no_config"})
		return
	}
	if err != nil {
		writeLocalizedError(w, r, http.StatusInternalServerError, err)
		return
	}
	var buf bytes.Buffer
	if err := configexport.Write(&buf, cfg, requestLang(r), configexport.Style{}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+configexport.Filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
