// Package api provides the embedded HTTP API and web dashboard server.
package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/update"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	assets "github.com/vpn-kongtrol/kongtrol/web"
)

// Server is the embedded HTTP API and dashboard server.
type Server struct {
	bind            string
	port            int
	adapters        *vpn.Set
	collector       *monitor.Collector
	routes          routing.RouteManager
	ks              security.KillSwitch
	killSwitchOn    atomic.Bool
	leakTest        *security.LeakTester
	policyEngine    atomic.Pointer[policy.Engine] // hot-swapped on policy CRUD; see saveRuntimeConfig
	policyResolver  *monitor.PolicyResolver
	configPath      string
	preferencesPath string
	onPolicyUpdate  func(*config.Config, *policy.Engine)
	// onSecurityToggle is invoked after a kill switch / DNS guard toggle
	// endpoint persists cfg, so the daemon can immediately re-apply live
	// enforcement (firewall rules / DNS override) — onPolicyUpdate only
	// rebuilds the service objects, it does not re-run Apply().
	onSecurityToggle  func(*config.Config)
	dnsMgr            *monitor.DNSManager
	dnsGuardOn        atomic.Bool
	connectMu         sync.Mutex
	connectCancel     map[string]context.CancelFunc
	configMu          sync.Mutex
	connectProfile    func(context.Context, string) error
	disconnectProfile func(context.Context, string) error
	apiToken          string
	sessions          sessionAuth // browser sessions + one-time nonces (session.go)
	upgrader          websocket.Upgrader
	srv               *http.Server
	onShutdown        func()
	// helperInstall backs POST /api/v1/helper/install; version (this
	// build's version) backs GET /api/v1/version and the helper status.
	// See SetHelperInstaller.
	helperInstall func(context.Context) error
	version       string
	// ksStatus / events back the split-tunnel kill switch state and the
	// policy notices; see SetKillSwitchStatus / SetEventLog.
	ksStatus func() security.KillSwitchStatus
	events   *monitor.EventLog
	// auditVerify checks an audit entry's HMAC (SetAuditVerifier); the
	// key never reaches the API.
	auditVerify func(security.AuditEvent) bool
	// blockedList returns the prefixes blocked now (SetBlockedList).
	blockedList func() []string
	// importDir is where POST /api/v1/vpns/import stores files
	// ("" = ~/.kongtrol/configs; SetImportDir).
	importDir string
	// readProtected reads a root-only profile file with one OS permission
	// prompt (SetProtectedReader; nil = not offered on this OS).
	readProtected func(ctx context.Context, path string) ([]byte, error)
	// setupMode / restart back /api/v1/setup and /api/v1/restart; see
	// SetSetupMode and SetRestarter.
	setupMode atomic.Bool
	restart   func() error
	// updateLast / onUpdateToggle / purgeData back /api/v1/updates and
	// /api/v1/data/purge; see handlers_lifecycle.go.
	updateLast     func() update.Result
	onUpdateToggle func(bool)
	purgeData      func() error
	// profileSync applies saved profiles live (profile_sync.go).
	profileSync func(*config.Config) ProfileSyncResult
	// recent backs GET /api/v1/connections/recent (handlers_connections.go).
	recent *monitor.RecentConnections
}

// NewServer creates a new API server.
func NewServer(
	bind string,
	port int,
	adapters *vpn.Set,
	collector *monitor.Collector,
	routes routing.RouteManager,
	ks security.KillSwitch,
	killSwitchEnabled bool,
	leakTest *security.LeakTester,
	policyEngine *policy.Engine,
	policyResolver *monitor.PolicyResolver,
	configPath string,
	onPolicyUpdate func(*config.Config, *policy.Engine),
	onSecurityToggle func(*config.Config),
	dnsMgr *monitor.DNSManager,
	dnsGuardEnabled bool,
	connectProfile func(context.Context, string) error,
	disconnectProfile func(context.Context, string) error,
	apiToken string,
	onShutdown func(),
) *Server {
	s := &Server{
		bind:              bind,
		port:              port,
		adapters:          adapters,
		collector:         collector,
		routes:            routes,
		ks:                ks,
		leakTest:          leakTest,
		policyResolver:    policyResolver,
		configPath:        configPath,
		onPolicyUpdate:    onPolicyUpdate,
		onSecurityToggle:  onSecurityToggle,
		dnsMgr:            dnsMgr,
		connectProfile:    connectProfile,
		disconnectProfile: disconnectProfile,
		apiToken:          apiToken,
		onShutdown:        onShutdown,
		connectCancel:     make(map[string]context.CancelFunc),
	}
	s.upgrader.CheckOrigin = s.checkWSOrigin
	s.policyEngine.Store(policyEngine)
	s.killSwitchOn.Store(killSwitchEnabled)
	s.dnsGuardOn.Store(dnsGuardEnabled)
	return s
}

// Start launches the HTTP server. Returns immediately; call Shutdown() to stop.
func (s *Server) Start() error {
	if err := config.ValidateDashboardBind(s.bind); err != nil {
		return err
	}
	if s.apiToken == "" {
		return fmt.Errorf("api: empty control token")
	}

	mux := http.NewServeMux()

	// Browser session bootstrap (session.go, ADR-0039, ADR-0062).
	mux.HandleFunc("POST "+sessionNoncePath, s.handleSessionNonce)
	mux.HandleFunc("POST "+sessionPath, s.handleSessionCreate)
	mux.HandleFunc("DELETE "+sessionPath, s.handleSessionDelete)
	mux.HandleFunc("POST "+wsTicketPath, s.handleWSTicket)

	// REST API endpoints.
	mux.HandleFunc("GET /api/v1/tunnels", s.handleListTunnels)
	mux.HandleFunc("GET /api/v1/metrics/history", s.handleMetricsHistory)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/connect", s.handleConnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/cancel_connect", s.handleCancelConnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/disconnect", s.handleDisconnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/reload", s.handleReloadTunnel)
	mux.HandleFunc("GET /api/v1/tunnels/routes", s.handleAllTunnelRoutes)
	mux.HandleFunc("GET /api/v1/tunnels/{name}/routes", s.handleTunnelRoutes)
	mux.HandleFunc("GET /api/v1/routes", s.handleListRoutes)
	mux.HandleFunc("GET /api/v1/network/overview", s.handleNetworkOverview)
	mux.HandleFunc("GET /api/v1/security/status", s.handleSecurityStatus)
	mux.HandleFunc("GET /api/v1/security/blocked", s.handleSecurityBlocked)
	mux.HandleFunc("POST /api/v1/security/killswitch", s.handleToggleKillSwitch)
	mux.HandleFunc("POST /api/v1/security/dnsguard", s.handleToggleDNSGuard)
	mux.HandleFunc("POST /api/v1/security/publicip", s.handleTogglePublicIP)
	mux.HandleFunc("POST /api/v1/security/publicip/check", s.handleCheckPublicIP)
	mux.HandleFunc("GET /api/v1/vpns", s.handleListVPNs)
	mux.HandleFunc("GET /api/v1/vpns/detect", s.handleDetectVPNs)
	mux.HandleFunc("GET /api/v1/connections/recent", s.handleRecentConnections)
	mux.HandleFunc("POST /api/v1/vpns", s.handleCreateVPN)
	mux.HandleFunc("POST /api/v1/vpns/import", s.handleImportVPNFile)
	mux.HandleFunc("PUT /api/v1/vpns/{name}", s.handleUpdateVPN)
	mux.HandleFunc("DELETE /api/v1/vpns/{name}", s.handleDeleteVPN)
	mux.HandleFunc("PUT /api/v1/vpns/{name}/killswitch", s.handleSetProfileKillSwitch)
	mux.HandleFunc("GET /api/v1/scheduler/rules", s.handleListScheduleRules)
	mux.HandleFunc("POST /api/v1/scheduler/rules", s.handleCreateScheduleRule)
	mux.HandleFunc("PUT /api/v1/scheduler/rules/{name}", s.handleUpdateScheduleRule)
	mux.HandleFunc("DELETE /api/v1/scheduler/rules/{name}", s.handleDeleteScheduleRule)
	mux.HandleFunc("GET /api/v1/audit", s.handleAuditLog)
	mux.HandleFunc("GET /api/v1/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/v1/settings", s.handleUpdateSettings)
	mux.HandleFunc("GET /api/v1/preferences", s.handleGetPreferences)
	mux.HandleFunc("PUT /api/v1/preferences", s.handleUpdatePreferences)
	mux.HandleFunc("GET /api/v1/groups", s.handleListGroups)
	mux.HandleFunc("POST /api/v1/groups", s.handleCreateGroup)
	mux.HandleFunc("PUT /api/v1/groups/{name}", s.handleUpdateGroup)
	mux.HandleFunc("DELETE /api/v1/groups/{name}", s.handleDeleteGroup)
	mux.HandleFunc("POST /api/v1/groups/{name}/connect", s.handleConnectGroup)
	mux.HandleFunc("POST /api/v1/groups/{name}/disconnect", s.handleDisconnectGroup)
	mux.HandleFunc("POST /api/v1/groups/{name}/reload", s.handleReloadGroup)
	mux.HandleFunc("GET /api/v1/policies", s.handleListPolicies)
	mux.HandleFunc("GET /api/v1/policies/meta", s.handlePoliciesMeta)
	mux.HandleFunc("POST /api/v1/policies/reload", s.handlePolicyReload)
	mux.HandleFunc("POST /api/v1/policies", s.handleCreatePolicy)
	mux.HandleFunc("PUT /api/v1/policies/{name}", s.handleUpdatePolicy)
	mux.HandleFunc("DELETE /api/v1/policies/{name}", s.handleDeletePolicy)
	mux.HandleFunc("POST /api/v1/policies/test", s.handleTestPolicy)
	mux.HandleFunc("GET /api/v1/policies/explain", s.handleExplainPolicy)
	mux.HandleFunc("GET /api/v1/resolve", s.handleResolve)
	mux.HandleFunc("GET /api/v1/dns/resolve", s.handleDNSResolve)
	mux.HandleFunc("POST /api/v1/shutdown", s.handleShutdown)
	mux.HandleFunc("POST /api/v1/restart", s.handleRestart)
	mux.HandleFunc("GET /api/v1/setup", s.handleSetupStatus)
	mux.HandleFunc("GET /api/v1/version", s.handleVersion)
	mux.HandleFunc("GET /api/v1/helper/status", s.handleHelperStatus)
	mux.HandleFunc("POST /api/v1/helper/install", s.handleHelperInstall)
	mux.HandleFunc("GET /api/v1/doctor", s.handleDoctor)
	mux.HandleFunc("GET /api/v1/config/export", s.handleConfigExport)
	mux.HandleFunc("GET /api/v1/events", s.handleEvents)
	mux.HandleFunc("GET /api/v1/updates", s.handleGetUpdates)
	mux.HandleFunc("POST /api/v1/updates", s.handleSetUpdates)
	mux.HandleFunc("POST /api/v1/data/purge", s.handlePurgeData)

	// WebSocket live metrics feed.
	mux.HandleFunc(wsPath, s.handleWebSocket)

	// The dashboard (web/ui) at "/": every path no route above claimed
	// (ADR-0043). Pages never carry credentials: a browser gets its session
	// cookie only from POST /api/v1/session with a one-time nonce.
	uiFS, uiBuilt := assets.UI()
	mux.Handle("/", newUIHandler(uiFS, uiBuilt))

	addr := net.JoinHostPort(listenHost(s.bind), fmt.Sprintf("%d", s.port))
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      s.secureHandler(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("api: listen on %s: %w", addr, err)
	}
	go func() {
		if err := s.srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "api: server error: %v\n", err)
		}
	}()

	return nil
}

func (s *Server) secureHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", s.contentSecurityPolicy(r.Host))
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		// Pages check Host too, so a DNS-rebinding page on another name
		// cannot read them either.
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			if !s.validHost(r.Host) {
				http.Error(w, "forbidden host", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if !s.validHost(r.Host) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == sessionPath && r.Method == http.MethodPost:
			// The nonce exchange: the one-time nonce is the credential.
			// Only browsers call it, and a browser always sends Origin on
			// a POST; a missing Origin is not proof of same origin (OWASP
			// CSRF cheat sheet), so it is required here.
			if !s.validBrowserOrigin(r) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		case r.URL.Path == sessionNoncePath:
			// Only native clients holding the file token mint nonces; a
			// browser session cannot.
			if !s.tokenAuthenticated(r) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		case r.URL.Path == wsPath && !s.tokenAuthenticated(r):
			// A browser cannot add headers to a WebSocket: this
			// daemon's Origin, the cookie and a one-time ticket from
			// POST /api/v1/ws-ticket (ADR-0062).
			if !s.validBrowserOrigin(r) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
			if !s.wsTicketAuthenticated(r) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		case !s.authenticated(r):
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Origin") != "" && !s.validBrowserOrigin(r) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		if isMutation(r.Method) && isConfigMutation(r.URL.Path) {
			s.configMu.Lock()
			defer s.configMu.Unlock()
		}
		if isMutation(r.Method) && r.ContentLength != 0 {
			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
				http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		}
		next.ServeHTTP(w, r)
	})
}

// checkWSOrigin is the WebSocket upgrader's origin check (secureHandler
// already authenticated the request): a browser must come from this
// daemon's origin; a native client sends no Origin and the file token.
func (s *Server) checkWSOrigin(r *http.Request) bool {
	if r.Header.Get("Origin") == "" {
		return s.validHost(r.Host) && s.tokenAuthenticated(r)
	}
	return s.validBrowserOrigin(r)
}

// contentSecurityPolicy is the page CSP. connect-src names the WebSocket
// origin explicitly: CSP3 lets 'self' match ws: on the same host, but not
// every browser does (MDN; WebKit bug 201591). host is the request's Host,
// already checked to be loopback on this port (any other gets 401/403).
func (s *Server) contentSecurityPolicy(host string) string {
	connect := "'self'"
	if s.validHost(host) {
		connect += " ws://" + host
	}
	return "default-src 'self'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' data:; connect-src " +
		connect + "; frame-ancestors 'none'"
}

// authenticated accepts the file token in a header (CLI, tray) or a live
// browser session: the cookie together with its page token in
// X-Kongtrol-Session (ADR-0062). The cookie alone, which any server on
// another port of 127.0.0.1 may have received, is not enough. The file
// token is never accepted from a cookie.
func (s *Server) authenticated(r *http.Request) bool {
	return s.tokenAuthenticated(r) || s.browserSession(r)
}

// tokenAuthenticated checks the file token in X-Kongtrol-Token or
// Authorization: Bearer, in constant time.
func (s *Server) tokenAuthenticated(r *http.Request) bool {
	token := strings.TrimSpace(r.Header.Get("X-Kongtrol-Token"))
	if token == "" {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	if token == "" || s.apiToken == "" {
		return false
	}
	return len(token) == len(s.apiToken) &&
		subtle.ConstantTimeCompare([]byte(token), []byte(s.apiToken)) == 1
}

func (s *Server) validBrowserOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" || u.Port() != fmt.Sprintf("%d", s.port) {
		return false
	}
	return isLoopbackHost(u.Hostname())
}

func (s *Server) validHost(hostport string) bool {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil || port != fmt.Sprintf("%d", s.port) {
		return false
	}
	return isLoopbackHost(host)
}

// listenHost is the address actually bound. "localhost" is pinned to
// 127.0.0.1: resolving the name could follow an edited hosts file to a
// non-loopback address (ValidateDashboardBind only allows loopback).
func listenHost(bind string) string {
	if strings.EqualFold(strings.TrimSuffix(strings.TrimSpace(bind), "."), "localhost") {
		return "127.0.0.1"
	}
	return bind
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isMutation(method string) bool {
	return method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodPatch || method == http.MethodDelete
}

func isConfigMutation(path string) bool {
	// The helper install waits minutes on an OS prompt; it must not hold
	// the config lock. Neither may the import, which can wait on one
	// (elevated read): it locks only to store the file.
	if strings.HasPrefix(path, "/api/v1/tunnels/") || strings.HasPrefix(path, "/api/v1/helper/") ||
		path == sessionPath || path == sessionNoncePath || path == wsTicketPath ||
		path == importPath || path == "/api/v1/shutdown" {
		return false
	}
	if strings.HasPrefix(path, "/api/v1/groups/") &&
		(strings.HasSuffix(path, "/connect") || strings.HasSuffix(path, "/disconnect")) {
		return false
	}
	return true
}

// Shutdown stops the HTTP server gracefully.
func (s *Server) Shutdown(ctx context.Context) error {
	s.connectMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.connectCancel))
	for name, cancel := range s.connectCancel {
		cancels = append(cancels, cancel)
		delete(s.connectCancel, name)
	}
	s.connectMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// Addr returns the full listening address (for display / browser open).
func (s *Server) Addr() string {
	return "http://" + net.JoinHostPort(s.bind, fmt.Sprintf("%d", s.port))
}

// adapterOf returns the adapter of profile name, or nil.
func (s *Server) adapterOf(name string) vpn.VPNAdapter {
	a, _ := s.adapters.Get(name)
	return a
}
