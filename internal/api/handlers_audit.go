package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// SetAuditVerifier sets the function that checks an audit entry's HMAC.
// The daemon keeps the signing key; the API only gets yes/no per entry.
func (s *Server) SetAuditVerifier(fn func(security.AuditEvent) bool) { s.auditVerify = fn }

// auditEntryDTO is one GET /api/v1/audit entry: the stored event plus
// whether it is signed and whether its signature checks out against the
// key in the OS keychain (false when unsigned, tampered or no key).
type auditEntryDTO struct {
	security.AuditEvent
	Signed   bool `json:"signed"`
	Verified bool `json:"verified"`
}

// auditMoreHeader is "true" when older matching entries remain: ask again
// with before=<timestamp of the last entry>.
const auditMoreHeader = "X-Kongtrol-Audit-More"

// GET /api/v1/audit?limit=&profile=&level=&from=&to=&before= — audit
// events, newest first, across every retained day (the same reader as
// `kongtrol logs`: security.QueryAuditEvents).
//
//   - limit: 1–1000 (default 200)
//   - profile: may repeat; an event of any of them matches
//   - from / to: a local day YYYY-MM-DD (to = end of that day) or RFC 3339
//   - before: RFC 3339 cursor, exclusive (the last timestamp of a page)
//
// The body stays a JSON array; the X-Kongtrol-Audit-More header says
// whether there is another page.
func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	q, err := parseAuditQuery(r)
	if err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, err)
		return
	}
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []auditEntryDTO{}
	more := false
	if cfg.Security.AuditLog.Path != "" {
		events, m, err := security.QueryAuditEvents(cfg.Security.AuditLog.Path, q)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		more = m
		for _, ev := range events {
			e := auditEntryDTO{AuditEvent: ev, Signed: ev.HMAC != ""}
			if e.Signed && s.auditVerify != nil {
				e.Verified = s.auditVerify(ev)
			}
			out = append(out, e)
		}
	}
	w.Header().Set(auditMoreHeader, strconv.FormatBool(more))
	writeJSON(w, http.StatusOK, out)
}

func parseAuditQuery(r *http.Request) (security.AuditQuery, error) {
	v := r.URL.Query()
	q := security.AuditQuery{
		Level: strings.TrimSpace(v.Get("level")),
		Limit: security.DefaultAuditQueryLimit,
	}
	// profile may repeat (the dashboard's multi-select): any of them matches.
	for _, p := range v["profile"] {
		if p = strings.TrimSpace(p); p != "" {
			q.Profiles = append(q.Profiles, p)
		}
	}
	if raw := strings.TrimSpace(v.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > security.MaxAuditQueryLimit {
			return q, &config.ValidationError{Key: "api.err.audit_limit", Args: []any{raw, security.MaxAuditQueryLimit}}
		}
		q.Limit = n
	}
	var err error
	if q.From, err = parseAuditTime(v.Get("from"), false); err != nil {
		return q, err
	}
	if q.To, err = parseAuditTime(v.Get("to"), true); err != nil {
		return q, err
	}
	if q.Before, err = parseAuditTime(v.Get("before"), false); err != nil {
		return q, err
	}
	if !q.From.IsZero() && !q.To.IsZero() && q.From.After(q.To) {
		return q, &config.ValidationError{Key: "api.err.audit_range"}
	}
	return q, nil
}

// parseAuditTime accepts a local day (YYYY-MM-DD; endOfDay picks its last
// instant) or an RFC 3339 time. Empty is the zero time (no bound).
func parseAuditTime(raw string, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if d, err := time.ParseInLocation("2006-01-02", raw, time.Local); err == nil {
		if endOfDay {
			return d.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
		}
		return d, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, nil
	}
	return time.Time{}, &config.ValidationError{Key: "api.err.audit_date", Args: []any{raw}}
}
