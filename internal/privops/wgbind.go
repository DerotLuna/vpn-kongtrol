package privops

import (
	"bufio"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync/atomic"

	"golang.org/x/crypto/curve25519"
)

// A tunnel Kongtrol did not start is bound to the caller's profile config
// by key, not by name: the public key derived from the config's
// [Interface] PrivateKey must equal the running interface's public key
// (`wg show <if> dump`, interface line field 2). A same-named tunnel of
// someone else (IT-managed, another app) has a different key, so a caller
// cannot read its counters, change its AllowedIPs or remove it.

// wgConfigPublicKey derives the base64 interface public key from a
// WireGuard config's [Interface] PrivateKey.
func wgConfigPublicKey(conf []byte) (string, bool) {
	sc := bufio.NewScanner(strings.NewReader(string(conf)))
	section := ""
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.IndexAny(line, "#;"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || section != "interface" || !strings.EqualFold(strings.TrimSpace(k), "PrivateKey") {
			continue
		}
		priv, err := base64.StdEncoding.DecodeString(strings.TrimSpace(v))
		if err != nil || len(priv) != curve25519.ScalarSize {
			return "", false
		}
		pub, err := curve25519.X25519(priv, curve25519.Basepoint)
		if err != nil {
			return "", false
		}
		return base64.StdEncoding.EncodeToString(pub), true
	}
	return "", false
}

// wgDumpPublicKey returns the interface public key from `wg show <if>
// dump` (first line: private-key, public-key, listen-port, fwmark).
func wgDumpPublicKey(dump string) (string, bool) {
	for _, line := range strings.Split(strings.ReplaceAll(dump, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 4 || ValidateWGKey(f[1]) != nil {
			return "", false
		}
		return f[1], true
	}
	return "", false
}

// wgKeyMatches reports whether the running tunnel name uses the key pair
// of conf. An unparsable key is a mismatch (deny). A dump failure is an
// error that never carries the command output (it may echo keys).
func wgKeyMatches(ctx context.Context, name string, conf []byte) (bool, error) {
	want, ok := wgConfigPublicKey(conf)
	if !ok {
		return false, nil
	}
	dump, err := wgDumpFn(ctx, name)
	if err != nil {
		return false, fmt.Errorf("%w: read WireGuard tunnel state", ErrFailed)
	}
	got, ok := wgDumpPublicKey(dump)
	return ok && subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1, nil
}

// ── audit mark ─────────────────────────────────────────────────────────

// The helper audits read-only ops only when they fail. An operation can
// ask for a success to be audited too (wg.stats of a tunnel authorized by
// its profile config rather than by Kongtrol having started it).

type auditMarkKey struct{}

// AuditMark is set by an operation that wants its success audited.
type AuditMark struct{ set atomic.Bool }

// Set reports whether the operation asked to be audited.
func (m *AuditMark) Set() bool { return m != nil && m.set.Load() }

// WithAuditMark attaches a fresh mark to ctx (the helper server does this
// per request).
func WithAuditMark(ctx context.Context) (context.Context, *AuditMark) {
	m := &AuditMark{}
	return context.WithValue(ctx, auditMarkKey{}, m), m
}

func markAudit(ctx context.Context) {
	if m, ok := ctx.Value(auditMarkKey{}).(*AuditMark); ok {
		m.set.Store(true)
	}
}
