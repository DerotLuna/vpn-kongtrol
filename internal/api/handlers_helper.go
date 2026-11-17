package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// helperInstallTimeout bounds one POST /api/v1/helper/install: the user
// has this long to answer the OS permission prompt.
const helperInstallTimeout = 3 * time.Minute

// Probes behind GET /api/v1/helper/status; tests replace them.
var (
	helperQueryState = privhelper.QueryStatus
	helperPing       = func(ctx context.Context) (privops.PingResult, error) {
		return privops.NewClient().Ping(ctx)
	}
	helperIsElevated = privops.IsElevated
)

// HelperStatus is the body of GET /api/v1/helper/status (and of a
// successful POST /api/v1/helper/install).
type HelperStatus struct {
	Installed     bool   `json:"installed"`
	Running       bool   `json:"running"`
	Available     bool   `json:"available"` // answers ping with our protocol version
	Elevated      bool   `json:"elevated"`  // the daemon itself is admin/root: no helper needed
	HelperVersion string `json:"helper_version"`
	CLIVersion    string `json:"cli_version"`
	NeedsInstall  bool   `json:"needs_install"`
}

// SetHelperInstaller enables POST /api/v1/helper/install. fn installs the
// privileged helper (one OS permission prompt) and returns a translated
// error; wrap privhelper.ErrElevationCancelled when the user declined.
// version is reported as cli_version. Call it before Start.
func (s *Server) SetHelperInstaller(version string, fn func(context.Context) error) {
	s.helperVersion = version
	s.helperInstall = fn
}

func (s *Server) helperStatus(ctx context.Context) HelperStatus {
	st := helperQueryState()
	out := HelperStatus{
		Installed:  st.Installed,
		Running:    st.Running,
		Elevated:   helperIsElevated(),
		CLIVersion: s.helperVersion,
	}
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if res, err := helperPing(pctx); err == nil {
		out.Installed, out.Running = true, true
		out.HelperVersion = res.Version
		out.Available = res.Protocol == privops.ProtocolVersion
	}
	out.NeedsInstall = !out.Elevated && !out.Available
	return out
}

func (s *Server) handleHelperStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.helperStatus(r.Context()))
}

// handleHelperInstall runs the installer (the OS shows one permission
// prompt), then checks the helper answers. Errors are translated:
// {"error", "hint", "error_key"} (+ "detail" with the raw cause).
func (s *Server) handleHelperInstall(w http.ResponseWriter, r *http.Request) {
	if s.helperInstall == nil {
		writeHelperError(w, r, http.StatusNotImplemented,
			"helper.install.unsupported", "helper.install.unsupported_hint", nil)
		return
	}
	// The user may take minutes to answer the prompt; outlive the
	// server's default WriteTimeout. Unsupported writers just ignore it.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(helperInstallTimeout + 30*time.Second))

	if st := s.helperStatus(r.Context()); !st.NeedsInstall {
		writeJSON(w, http.StatusOK, st)
		return
	}
	// A client that gives up must not kill the prompt half-way.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), helperInstallTimeout)
	defer cancel()
	if err := s.helperInstall(ctx); err != nil {
		switch {
		case errors.Is(err, privhelper.ErrElevationCancelled):
			writeHelperError(w, r, http.StatusForbidden,
				"service.elevation.cancelled", "helper.install.cancelled_hint", nil)
		case ctx.Err() != nil:
			writeHelperError(w, r, http.StatusGatewayTimeout,
				"helper.install.timeout", "helper.install.timeout_hint", err)
		default:
			writeHelperError(w, r, http.StatusInternalServerError,
				"service.install.failed", "helper.install.failed_hint", err)
		}
		return
	}
	st := s.helperStatus(ctx)
	if st.NeedsInstall {
		writeHelperError(w, r, http.StatusInternalServerError,
			"service.install.no_response", "helper.install.no_response_hint", nil)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// writeHelperError answers a translated failure plus one "what to do" line,
// in the shape of writeLocalizedError.
func writeHelperError(w http.ResponseWriter, r *http.Request, status int, key, hintKey string, cause error) {
	lang := requestLang(r)
	body := map[string]string{
		"error":     i18n.T(lang, key),
		"hint":      i18n.T(lang, hintKey),
		"error_key": key,
	}
	if cause != nil {
		body["detail"] = cause.Error()
	}
	writeJSON(w, status, body)
}
