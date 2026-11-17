package wireguard

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type recordOps struct {
	privops.Ops
	remove privops.WGRemoveRequest
}

func (r *recordOps) WGRemoveTunnel(_ context.Context, req privops.WGRemoveRequest) error {
	r.remove = req
	return nil
}

// Disconnect sends the profile .conf (never the deleted patched temp copy)
// so the helper can adopt a tunnel started by an older Kongtrol.
func TestDisconnect_SendsProfileConfig(t *testing.T) {
	rec := &recordOps{}
	old := opsFunc
	defer func() { opsFunc = old }()
	opsFunc = func(context.Context) (privops.Ops, error) { return rec, nil }

	profile := filepath.Join(t.TempDir(), "rluna.conf")
	a := &Adapter{}
	a.Configure(vpn.AdapterConfig{ConfigPath: profile})
	// A later connect through a patched copy must not replace it.
	a.mu.Lock()
	if tmp := filepath.Join(t.TempDir(), "kongtrol-wg-123", "rluna.conf"); !isPatchedCopy(tmp) {
		t.Fatal("patched copy not recognised")
	}
	a.mu.Unlock()
	a.Configure(vpn.AdapterConfig{ConfigPath: filepath.Join(t.TempDir(), "kongtrol-wg-9", "rluna.conf")})

	if err := a.Disconnect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if rec.remove.Name != "rluna" || rec.remove.ConfigPath != profile {
		t.Fatalf("remove request %+v, want name rluna config %s", rec.remove, profile)
	}
}
