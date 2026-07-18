// Package api provides the embedded HTTP API and web dashboard server.
package api

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
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
	bind           string
	port           int
	adapters       map[string]vpn.VPNAdapter
	collector      *monitor.Collector
	routes         routing.RouteManager
	ks             security.KillSwitch
	killSwitchOn   atomic.Bool
	leakTest       *security.LeakTester
	policyEngine   atomic.Pointer[policy.Engine] // hot-swapped on policy CRUD; see saveRuntimeConfig
	policyResolver *monitor.PolicyResolver
	configPath     string
	onPolicyUpdate func(*config.Config, *policy.Engine)
	// onSecurityToggle is invoked after a kill switch / DNS guard toggle
	// endpoint persists cfg, so the daemon can immediately re-apply live
	// enforcement (firewall rules / DNS override) — onPolicyUpdate only
	// rebuilds the service objects, it does not re-run Apply().
	onSecurityToggle func(*config.Config)
	dnsMgr           *monitor.DNSManager
	dnsGuardOn       atomic.Bool
	connectMu        sync.Mutex
	connectCancel    map[string]context.CancelFunc
	upgrader         websocket.Upgrader
	srv              *http.Server
	onShutdown       func()
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
	onShutdown func(),
) *Server {
	s := &Server{
		bind:             bind,
		port:             port,
		adapters:         adapters,
		collector:        collector,
		routes:           routes,
		ks:               ks,
		leakTest:         leakTest,
		policyResolver:   policyResolver,
		configPath:       configPath,
		onPolicyUpdate:   onPolicyUpdate,
		onSecurityToggle: onSecurityToggle,
		dnsMgr:           dnsMgr,
		onShutdown:       onShutdown,
		connectCancel:    make(map[string]context.CancelFunc),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Only allow connections from localhost.
				return r.Host == fmt.Sprintf("%s:%d", bind, port) ||
					r.Host == fmt.Sprintf("localhost:%d", port) ||
					r.Host == fmt.Sprintf("127.0.0.1:%d", port)
			},
		},
	}
	s.policyEngine.Store(policyEngine)
	s.killSwitchOn.Store(killSwitchEnabled)
	s.dnsGuardOn.Store(dnsGuardEnabled)
	return s
}

// Start launches the HTTP server. Returns immediately; call Shutdown() to stop.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// REST API endpoints.
	mux.HandleFunc("GET /api/v1/tunnels", s.handleListTunnels)
	mux.HandleFunc("GET /api/v1/metrics/history", s.handleMetricsHistory)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/connect", s.handleConnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/cancel_connect", s.handleCancelConnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/disconnect", s.handleDisconnect)
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
	mux.HandleFunc("GET /api/v1/groups", s.handleListGroups)
	mux.HandleFunc("POST /api/v1/groups", s.handleCreateGroup)
	mux.HandleFunc("PUT /api/v1/groups/{name}", s.handleUpdateGroup)
	mux.HandleFunc("DELETE /api/v1/groups/{name}", s.handleDeleteGroup)
	mux.HandleFunc("POST /api/v1/groups/{name}/connect", s.handleConnectGroup)
	mux.HandleFunc("POST /api/v1/groups/{name}/disconnect", s.handleDisconnectGroup)
	mux.HandleFunc("GET /api/v1/policies", s.handleListPolicies)
	mux.HandleFunc("GET /api/v1/policies/meta", s.handlePoliciesMeta)
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
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		fileServer.ServeHTTP(w, r)
	}))

	addr := fmt.Sprintf("%s:%d", s.bind, s.port)
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("api: server error: %v\n", err)
		}
	}()

	return nil
}

// Shutdown stops the HTTP server gracefully.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// Addr returns the full listening address (for display / browser open).
func (s *Server) Addr() string {
	return fmt.Sprintf("http://%s:%d", s.bind, s.port)
}
