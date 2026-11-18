package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// doJSONLang is doJSON with the dashboard's language header.
func doJSONLang(t *testing.T, s *Server, method, target, pattern string, h http.HandlerFunc, body, lang string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(method+" "+pattern, h)
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("X-Kongtrol-Lang", lang)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

type localizedErr struct {
	Error    string `json:"error"`
	Hint     string `json:"hint"`
	ErrorKey string `json:"error_key"`
}

func decodeLocalized(t *testing.T, rec *httptest.ResponseRecorder) localizedErr {
	t.Helper()
	var e localizedErr
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("body %s: %v", rec.Body, err)
	}
	if e.Error == "" || e.Hint == "" || e.ErrorKey == "" {
		t.Fatalf("not a localized error: %s", rec.Body)
	}
	return e
}

// Groups and scheduler rules answer every user error with a translated
// message, a "what to do" line and a stable key (never raw English only).
func TestGroupsAndSchedule_LocalizedErrors(t *testing.T) {
	cases := []struct {
		name, method, target, pattern, body string
		h                                   func(*Server) http.HandlerFunc
		status                              int
		key                                 string
	}{
		{"group bad json", http.MethodPost, "/api/v1/groups", "/api/v1/groups", `{`,
			func(s *Server) http.HandlerFunc { return s.handleCreateGroup }, 400, "api.err.invalid_json"},
		{"group no name", http.MethodPost, "/api/v1/groups", "/api/v1/groups", `{"name":" ","profiles":["office"]}`,
			func(s *Server) http.HandlerFunc { return s.handleCreateGroup }, 400, "api.err.name_required"},
		{"group exists", http.MethodPost, "/api/v1/groups", "/api/v1/groups", `{"name":"work","profiles":["office"]}`,
			func(s *Server) http.HandlerFunc { return s.handleCreateGroup }, 409, "api.err.group_exists"},
		{"group unknown profile", http.MethodPost, "/api/v1/groups", "/api/v1/groups", `{"name":"x","profiles":["ghost"]}`,
			func(s *Server) http.HandlerFunc { return s.handleCreateGroup }, 400, "config.err.group_unknown_profile"},
		{"group empty", http.MethodPost, "/api/v1/groups", "/api/v1/groups", `{"name":"x","profiles":[" "]}`,
			func(s *Server) http.HandlerFunc { return s.handleCreateGroup }, 400, "config.err.group_empty"},
		{"group not found", http.MethodPut, "/api/v1/groups/nope", "/api/v1/groups/{name}", `{}`,
			func(s *Server) http.HandlerFunc { return s.handleUpdateGroup }, 404, "api.err.group_not_found"},
		{"rule exists", http.MethodPost, "/api/v1/scheduler/rules", "/api/v1/scheduler/rules", `{"name":"Weekdays","profiles":["office"]}`,
			func(s *Server) http.HandlerFunc { return s.handleCreateScheduleRule }, 409, "api.err.rule_exists"},
		{"rule bad time", http.MethodPost, "/api/v1/scheduler/rules", "/api/v1/scheduler/rules", `{"name":"n","profiles":["office"],"start":"9:00"}`,
			func(s *Server) http.HandlerFunc { return s.handleCreateScheduleRule }, 400, "config.err.rule_time"},
		{"rule bad day", http.MethodPost, "/api/v1/scheduler/rules", "/api/v1/scheduler/rules", `{"name":"n","profiles":["office"],"weekdays":["lunes"]}`,
			func(s *Server) http.HandlerFunc { return s.handleCreateScheduleRule }, 400, "config.err.rule_weekday"},
		{"rule unknown profile", http.MethodPost, "/api/v1/scheduler/rules", "/api/v1/scheduler/rules", `{"name":"n","profiles":["ghost"]}`,
			func(s *Server) http.HandlerFunc { return s.handleCreateScheduleRule }, 400, "config.err.rule_unknown_profile"},
		{"rule not found", http.MethodDelete, "/api/v1/scheduler/rules/nope", "/api/v1/scheduler/rules/{name}", ``,
			func(s *Server) http.HandlerFunc { return s.handleDeleteScheduleRule }, 404, "api.err.rule_not_found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := newReloadTestServer(t, mergeTestYAML, nil)
			rec := doJSONLang(t, s, c.method, c.target, c.pattern, c.h(s), c.body, "es")
			if rec.Code != c.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, c.status, rec.Body)
			}
			e := decodeLocalized(t, rec)
			if e.ErrorKey != c.key {
				t.Fatalf("error_key %q, want %q (%s)", e.ErrorKey, c.key, e.Error)
			}
		})
	}
}

// A rename onto another rule's name is a conflict, not a silent duplicate.
func TestUpdateScheduleRule_RenameOntoExistingConflicts(t *testing.T) {
	s, _ := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodPost, "/api/v1/scheduler/rules", "/api/v1/scheduler/rules", s.handleCreateScheduleRule,
		`{"name":"nights","profiles":["backup"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	rec = doJSON(t, s, http.MethodPut, "/api/v1/scheduler/rules/nights", "/api/v1/scheduler/rules/{name}", s.handleUpdateScheduleRule,
		`{"name":"weekdays"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, want 409: %s", rec.Code, rec.Body)
	}
}
