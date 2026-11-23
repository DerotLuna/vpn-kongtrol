package api

import (
	"net/http"
	"sync"
	"time"
)

// Audit of the local API (ADR-0064). Every request that changes something
// (POST/PUT/DELETE) writes one audit entry with who asked (the app's file
// token or a dashboard session), what (the action of the route and the
// name in its path), when (the entry's time) and the result (HTTP status).
// Like Kubernetes' "Metadata" audit level, bodies are never read or logged,
// so passwords, keys and file contents cannot reach the log. Refused
// credentials are logged too, throttled.

// AuditRecord is one audited API request; the daemon renders and writes it
// (SetAuditLog), the API stays free of i18n and of the logger.
type AuditRecord struct {
	Level  string // INFO | WARN | SECURITY
	Action string // e.g. "config.profile.update", "auth.denied"
	// Profile is the VPN profile in the path ({name} of /tunnels and
	// /vpns routes); Name is any other {name} (policy, group, rule).
	Profile string
	Name    string
	// Client is "app" (file token: CLI, tray), "dashboard" (browser
	// session cookie) or "unknown" (no valid credential).
	Client string
	Status int
	// Reason (auth.denied only): "credential", "origin" or "host";
	// Suppressed counts the refusals not logged before this one.
	Reason     string
	Method     string
	Path       string
	Suppressed int
}

// SetAuditLog sets where audited requests go (nil = not audited).
func (s *Server) SetAuditLog(fn func(AuditRecord)) { s.auditLog = fn }

type auditRoute struct {
	action string
	level  string // on success; a refused request is WARN
	// profile: the {name} path value is a VPN profile.
	profile bool
}

// auditRoutes maps every mutating route pattern (as registered in Start)
// to its audit action. A test checks that no mutating route is missing.
// Read-only POSTs (policies/test) are listed in auditSkip.
var auditRoutes = map[string]auditRoute{
	"POST " + sessionNoncePath: {action: "auth.nonce", level: "INFO"},
	"POST " + sessionPath:      {action: "auth.session_create", level: "SECURITY"},
	"DELETE " + sessionPath:    {action: "auth.session_end", level: "INFO"},

	"POST /api/v1/tunnels/{name}/connect":        {action: "api.tunnel.connect", level: "INFO", profile: true},
	"POST /api/v1/tunnels/{name}/cancel_connect": {action: "api.tunnel.cancel_connect", level: "INFO", profile: true},
	"POST /api/v1/tunnels/{name}/disconnect":     {action: "api.tunnel.disconnect", level: "INFO", profile: true},
	"POST /api/v1/tunnels/{name}/reload":         {action: "api.tunnel.reload", level: "INFO", profile: true},
	"POST /api/v1/groups/{name}/connect":         {action: "api.group.connect", level: "INFO"},
	"POST /api/v1/groups/{name}/disconnect":      {action: "api.group.disconnect", level: "INFO"},
	"POST /api/v1/groups/{name}/reload":          {action: "api.group.reload", level: "INFO"},

	"POST /api/v1/security/killswitch":       {action: "security.killswitch.set", level: "SECURITY"},
	"POST /api/v1/security/dnsguard":         {action: "security.dnsguard.set", level: "SECURITY"},
	"POST /api/v1/security/publicip":         {action: "security.publicip.set", level: "SECURITY"},
	"POST /api/v1/security/publicip/check":   {action: "security.publicip.check", level: "INFO"},
	"PUT /api/v1/vpns/{name}/killswitch":     {action: "security.profile_killswitch.set", level: "SECURITY", profile: true},
	"PUT /api/v1/vpns/{name}/default-egress": {action: "vpn.default_egress.set", level: "SECURITY", profile: true},

	"POST /api/v1/vpns":          {action: "config.profile.create", level: "SECURITY"},
	"POST /api/v1/vpns/import":   {action: "config.profile.import", level: "SECURITY"},
	"PUT /api/v1/vpns/{name}":    {action: "config.profile.update", level: "SECURITY", profile: true},
	"DELETE /api/v1/vpns/{name}": {action: "config.profile.delete", level: "SECURITY", profile: true},

	"POST /api/v1/policies":                 {action: "config.policy.create", level: "INFO"},
	"PUT /api/v1/policies/{name}":           {action: "config.policy.update", level: "INFO"},
	"DELETE /api/v1/policies/{name}":        {action: "config.policy.delete", level: "INFO"},
	"POST /api/v1/policies/reload":          {action: "config.policy.reload", level: "INFO"},
	"POST /api/v1/policies/move":            {action: "config.policy.move", level: "INFO"},
	"POST /api/v1/groups":                   {action: "config.group.create", level: "INFO"},
	"PUT /api/v1/groups/{name}":             {action: "config.group.update", level: "INFO"},
	"DELETE /api/v1/groups/{name}":          {action: "config.group.delete", level: "INFO"},
	"POST /api/v1/scheduler/rules":          {action: "config.schedule.create", level: "INFO"},
	"PUT /api/v1/scheduler/rules/{name}":    {action: "config.schedule.update", level: "INFO"},
	"DELETE /api/v1/scheduler/rules/{name}": {action: "config.schedule.delete", level: "INFO"},
	"PUT /api/v1/settings":                  {action: "config.settings.update", level: "SECURITY"},
	"PUT /api/v1/preferences":               {action: "config.preferences.update", level: "INFO"},
	"POST /api/v1/updates":                  {action: "config.update_check.set", level: "INFO"},
	"PUT /api/v1/startup":                   {action: "config.startup.set", level: "INFO"},

	"POST /api/v1/shutdown":       {action: "daemon.shutdown", level: "INFO"},
	"POST /api/v1/restart":        {action: "daemon.restart_request", level: "INFO"},
	"POST /api/v1/helper/install": {action: "helper.install", level: "SECURITY"},
	// The purge deletes the audit log itself; the entry is written before
	// it runs (see auditBefore) and the daemon log keeps the result.
	"POST /api/v1/data/purge": {action: "data.purge", level: "SECURITY"},
}

// auditSkip are mutating-method routes that change nothing.
var auditSkip = map[string]bool{
	"POST /api/v1/policies/test": true,
}

// auditBefore routes are logged when the request arrives, because the
// handler removes the log they would be written to.
var auditBefore = map[string]bool{
	"POST /api/v1/data/purge": true,
}

// clientOf tells who sent an authenticated request.
func (s *Server) clientOf(r *http.Request) string {
	switch {
	case s.tokenAuthenticated(r):
		return "app"
	case r.Header.Get("Origin") != "" || hasCookie(r, sessionCookieName):
		return "dashboard"
	}
	return "unknown"
}

func hasCookie(r *http.Request, name string) bool {
	_, err := r.Cookie(name)
	return err == nil
}

// statusRecorder keeps the status code the handler wrote.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the real writer.
func (w *statusRecorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// serveAudited runs next and writes the audit entry of a mutating request.
func (s *Server) serveAudited(next http.Handler, w http.ResponseWriter, r *http.Request) {
	if s.auditLog == nil || !isMutation(r.Method) {
		next.ServeHTTP(w, r)
		return
	}
	client := s.clientOf(r)
	rec := &statusRecorder{ResponseWriter: w}
	// The pattern is only known after the mux matched; the purge is
	// recognised by its fixed path to log it first.
	if key := r.Method + " " + r.URL.Path; auditBefore[key] {
		s.writeAudit(r, auditRoutes[key], key, client, 0)
	}
	next.ServeHTTP(rec, r)
	key := r.Pattern
	if key == "" || auditSkip[key] || auditBefore[key] {
		return
	}
	status := rec.status
	if status == 0 {
		status = http.StatusOK
	}
	route, ok := auditRoutes[key]
	if !ok {
		route = auditRoute{action: "api.request", level: "INFO"}
	}
	s.writeAudit(r, route, key, client, status)
}

func (s *Server) writeAudit(r *http.Request, route auditRoute, key, client string, status int) {
	rec := AuditRecord{
		Level: route.level, Action: route.action, Client: client, Status: status,
		Method: r.Method, Path: clip(r.URL.Path),
	}
	if status >= 400 {
		rec.Level = "WARN"
	}
	if name := r.PathValue("name"); name != "" {
		if route.profile {
			rec.Profile = name
		} else {
			rec.Name = name
		}
	}
	if route.action == "api.request" {
		rec.Name = key
	}
	s.auditLog(rec)
}

// authAudit throttles the "access refused" entries: a local program
// retrying with a wrong token must not fill the log. At most one entry per
// window; the next one says how many were skipped.
type authAudit struct {
	mu         sync.Mutex
	last       time.Time
	suppressed int
	now        func() time.Time
}

const authAuditWindow = 10 * time.Second

// auditDenied logs a refused request (no valid credential, foreign origin
// or host), throttled.
func (s *Server) auditDenied(r *http.Request, reason string, status int) {
	if s.auditLog == nil {
		return
	}
	a := &s.authAudit
	a.mu.Lock()
	now := time.Now()
	if a.now != nil {
		now = a.now()
	}
	if !a.last.IsZero() && now.Sub(a.last) < authAuditWindow {
		a.suppressed++
		a.mu.Unlock()
		return
	}
	skipped := a.suppressed
	a.last, a.suppressed = now, 0
	a.mu.Unlock()
	s.auditLog(AuditRecord{
		Level: "SECURITY", Action: "auth.denied", Client: "unknown", Status: status,
		Reason: reason, Method: r.Method, Path: clip(r.URL.Path), Suppressed: skipped,
	})
}

// clip bounds a request path copied into the log (a refused request's
// path is chosen by whoever sent it).
func clip(p string) string {
	const maxLen = 200
	if len(p) > maxLen {
		return p[:maxLen] + "…"
	}
	return p
}
