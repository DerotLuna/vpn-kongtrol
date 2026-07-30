// Package api provides the embedded HTTP API and web dashboard server.
package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io/fs"
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
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	assets "github.com/vpn-kongtrol/kongtrol/web"
)

// Server is the embedded HTTP API and dashboard server.
type Server struct {
	bind            string
	port            int
	adapters        map[string]vpn.VPNAdapter
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
	upgrader          websocket.Upgrader
	srv               *http.Server
	onShutdown        func()
}

// NewServer creates a new API server.
func NewServer(
	bind string,
	port int,
	adapters map[string]vpn.VPNAdapter,
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
	s.upgrader.CheckOrigin = func(r *http.Request) bool {
		if r.Header.Get("Origin") == "" {
			return s.validHost(r.Host) && s.authenticated(r)
		}
		return s.validBrowserOrigin(r)
	}
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

	// REST API endpoints.
	mux.HandleFunc("GET /api/v1/tunnels", s.handleListTunnels)
	mux.HandleFunc("GET /api/v1/metrics/history", s.handleMetricsHistory)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/connect", s.handleConnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/cancel_connect", s.handleCancelConnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/disconnect", s.handleDisconnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/reload", s.handleReloadTunnel)
	mux.HandleFunc("GET /api/v1/routes", s.handleListRoutes)
	mux.HandleFunc("GET /api/v1/network/overview", s.handleNetworkOverview)
	mux.HandleFunc("GET /api/v1/security/status", s.handleSecurityStatus)
	mux.HandleFunc("POST /api/v1/security/killswitch", s.handleToggleKillSwitch)
	mux.HandleFunc("POST /api/v1/security/dnsguard", s.handleToggleDNSGuard)
	mux.HandleFunc("GET /api/v1/vpns", s.handleListVPNs)
	mux.HandleFunc("POST /api/v1/vpns", s.handleCreateVPN)
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
	mux.HandleFunc("GET /api/v1/resolve", s.handleResolve)
	mux.HandleFunc("GET /api/v1/dns/resolve", s.handleDNSResolve)
	mux.HandleFunc("POST /api/v1/shutdown", s.handleShutdown)

	// WebSocket live metrics feed.
	mux.HandleFunc("/api/v1/ws/metrics", s.handleWebSocket)

	// Embedded dashboard — served at / with no-cache to pick up new versions.
	dashFS, err := fs.Sub(assets.FS, "dashboard")
	if err != nil {
		return fmt.Errorf("api: embed dashboard: %w", err)
	}
	fileServer := http.FileServer(http.FS(dashFS))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "kongtrol_control",
			Value:    s.apiToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		fileServer.ServeHTTP(w, r)
	}))

	addr := net.JoinHostPort(s.bind, fmt.Sprintf("%d", s.port))
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if !s.validHost(r.Host) || !s.authenticated(r) {
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

func (s *Server) authenticated(r *http.Request) bool {
	token := strings.TrimSpace(r.Header.Get("X-Kongtrol-Token"))
	if token == "" {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	if token == "" {
		if cookie, err := r.Cookie("kongtrol_control"); err == nil {
			token = cookie.Value
		}
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
	if strings.HasPrefix(path, "/api/v1/tunnels/") || path == "/api/v1/shutdown" {
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
