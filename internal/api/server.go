// Package api provides the embedded HTTP API and web dashboard server.
package api

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
	killSwitchOn   bool
	leakTest       *security.LeakTester
	policyEngine   *policy.Engine
	policyResolver *monitor.PolicyResolver
	dnsMgr         *monitor.DNSManager
	dnsGuardOn     bool
	connectMu      sync.Mutex
	connectCancel  map[string]context.CancelFunc
	upgrader       websocket.Upgrader
	srv            *http.Server
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
	dnsMgr *monitor.DNSManager,
	dnsGuardEnabled bool,
) *Server {
	return &Server{
		bind:           bind,
		port:           port,
		adapters:       adapters,
		collector:      collector,
		routes:         routes,
		ks:             ks,
		killSwitchOn:   killSwitchEnabled,
		leakTest:       leakTest,
		policyEngine:   policyEngine,
		policyResolver: policyResolver,
		dnsMgr:         dnsMgr,
		dnsGuardOn:     dnsGuardEnabled,
		connectCancel:  make(map[string]context.CancelFunc),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Only allow connections from localhost.
				return r.Host == fmt.Sprintf("%s:%d", bind, port) ||
					r.Host == fmt.Sprintf("localhost:%d", port) ||
					r.Host == fmt.Sprintf("127.0.0.1:%d", port)
			},
		},
	}
}

// Start launches the HTTP server. Returns immediately; call Shutdown() to stop.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// REST API endpoints.
	mux.HandleFunc("GET /api/v1/tunnels", s.handleListTunnels)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/connect", s.handleConnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/cancel_connect", s.handleCancelConnect)
	mux.HandleFunc("POST /api/v1/tunnels/{name}/disconnect", s.handleDisconnect)
	mux.HandleFunc("GET /api/v1/routes", s.handleListRoutes)
	mux.HandleFunc("GET /api/v1/network/overview", s.handleNetworkOverview)
	mux.HandleFunc("GET /api/v1/security/status", s.handleSecurityStatus)
	mux.HandleFunc("GET /api/v1/policies", s.handleListPolicies)
	mux.HandleFunc("GET /api/v1/resolve", s.handleResolve)

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
