package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// GET /api/v1/audit marks each entry signed/verified with the daemon's
// verifier (the key never reaches the API), pages with before= and says
// whether more remain; bad dates answer a translated 400.
func TestAuditLog_VerifiedPagingAndDates(t *testing.T) {
	dir := t.TempDir()
	base := filepath.ToSlash(filepath.Join(dir, "audit.log"))
	yaml := strings.Replace(baseTestYAML, "vpns:", "security:\n  audit_log:\n    path: "+base+"\n    sign: true\nvpns:", 1)
	s, _ := newReloadTestServer(t, yaml, nil)

	key := []byte("0123456789abcdef0123456789abcdef")
	l, err := security.NewAuditLogger(base, security.AuditLoggerOptions{Sign: true, Key: key})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		l.Info("vpn.connect", "office", "n")
		time.Sleep(2 * time.Millisecond)
	}
	l.Info("vpn.connect", "home", "n")
	time.Sleep(2 * time.Millisecond)
	l.Info("vpn.connect", "cafe", "n")
	_ = l.Close()
	// One entry edited after it was written.
	tampered := ""
	s.SetAuditVerifier(func(ev security.AuditEvent) bool {
		if tampered == "" {
			tampered = ev.ID // the newest one
		}
		return ev.ID != tampered && l.Verify(ev)
	})

	get := func(q string) ([]auditEntryDTO, *http.Response) {
		rec := doJSONLang(t, s, http.MethodGet, "/api/v1/audit"+q, "/api/v1/audit", s.handleAuditLog, "", "en")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", q, rec.Code, rec.Body)
		}
		var out []auditEntryDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out, rec.Result()
	}

	page, res := get("?limit=3")
	if len(page) != 3 || res.Header.Get(auditMoreHeader) != "true" {
		t.Fatalf("page 1: %d entries, more=%q", len(page), res.Header.Get(auditMoreHeader))
	}
	if !page[0].Signed || page[0].Verified || !page[1].Verified {
		t.Fatalf("verification flags wrong: %+v", page[:2])
	}
	if strings.Contains(string(mustJSON(t, page)), string(key)) {
		t.Fatal("key leaked")
	}
	rest, res := get("?limit=4&before=" + page[2].Timestamp.Format(time.RFC3339Nano))
	if len(rest) != 4 || res.Header.Get(auditMoreHeader) != "false" {
		t.Fatalf("page 2: %d entries, more=%q", len(rest), res.Header.Get(auditMoreHeader))
	}

	// profile repeats: an entry of any of them matches (case-insensitive).
	if got, _ := get("?profile=HOME&profile=cafe&profile=%20"); len(got) != 2 {
		t.Fatalf("two profiles: %d entries", len(got))
	}
	if got, _ := get("?profile=office"); len(got) != 5 {
		t.Fatalf("one profile: %d entries", len(got))
	}

	today := time.Now().Format("2006-01-02")
	if got, _ := get("?from=" + today + "&to=" + today); len(got) != 7 {
		t.Fatalf("today range: %d", len(got))
	}
	if got, _ := get("?to=2001-01-01"); len(got) != 0 {
		t.Fatalf("old range: %d", len(got))
	}

	for q, key := range map[string]string{
		"?from=yesterday":                "api.err.audit_date",
		"?limit=0":                       "api.err.audit_limit",
		"?limit=5000":                    "api.err.audit_limit",
		"?from=2026-11-18&to=2026-11-01": "api.err.audit_range",
	} {
		rec := doJSONLang(t, s, http.MethodGet, "/api/v1/audit"+q, "/api/v1/audit", s.handleAuditLog, "", "es")
		if rec.Code != http.StatusBadRequest || decodeLocalized(t, rec).ErrorKey != key {
			t.Fatalf("%s: %d %s", q, rec.Code, rec.Body)
		}
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
