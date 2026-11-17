package wireguard

import (
	"context"
	"fmt"
	"net/netip"
	"path/filepath"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// privTimeout bounds one privileged WireGuard call (installing a tunnel
// service can take several seconds on Windows).
const privTimeout = 60 * time.Second

// opsFunc returns the privileged implementation (in-process when elevated,
// else the Kongtrol helper; ErrHelperNotInstalled when missing).
var opsFunc = privops.Default

func withOps(fn func(ctx context.Context, ops privops.Ops) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), privTimeout)
	defer cancel()
	ops, err := opsFunc(ctx)
	if err != nil {
		return err
	}
	return fn(ctx, ops)
}

// wgUp brings the tunnel up through privops: the helper reads and checks
// the user's .conf, keeps a private copy named <name>.conf and runs
// wireguard.exe /installtunnelservice (Windows) or wg-quick up (Unix) on it.
func wgUp(configPath, name string) error {
	if alreadyUp := wgPreUp(name); alreadyUp {
		return nil
	}
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}
	return withOps(func(ctx context.Context, ops privops.Ops) error {
		_, err := ops.WGInstallTunnel(ctx, privops.WGInstallRequest{ConfigPath: abs, Name: name})
		return err
	})
}

// wgDown stops and removes the tunnel. profileConfig (the profile .conf)
// lets the helper adopt a tunnel brought up by an older Kongtrol or the
// WireGuard app; see privops.WGRemoveRequest.
func wgDown(name, profileConfig string) error {
	if name == "" {
		return fmt.Errorf("wgDown: tunnel name is empty")
	}
	return withOps(func(ctx context.Context, ops privops.Ops) error {
		return ops.WGRemoveTunnel(ctx, privops.WGRemoveRequest{Name: name, ConfigPath: absOrEmpty(profileConfig)})
	})
}

// WgSetAllowedIPs updates AllowedIPs for a peer on a live WireGuard tunnel.
// On Windows the tunnel service also updates the OS routes; on Unix this
// only updates crypto-routing — the caller adds OS routes via routeMgr.Add().
func WgSetAllowedIPs(ifaceName, peerPubKey string, cidrs []string, profileConfig string) error {
	req := privops.WGAllowedIPsRequest{Interface: ifaceName, PeerPublicKey: peerPubKey, ConfigPath: absOrEmpty(profileConfig)}
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(strings.TrimSpace(c))
		if err != nil {
			return fmt.Errorf("wireguard: allowed IP %q: %w", c, err)
		}
		req.AllowedIPs = append(req.AllowedIPs, p)
	}
	return withOps(func(ctx context.Context, ops privops.Ops) error {
		return ops.WGSetAllowedIPs(ctx, req)
	})
}

// absOrEmpty makes the profile config absolute (the helper requires it).
func absOrEmpty(p string) string {
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// isPatchedCopy reports whether path is a temporary patched copy of a
// profile config (os.MkdirTemp "kongtrol-wg-*"), which is deleted after
// connect and so can't identify the profile later.
func isPatchedCopy(path string) bool {
	return strings.HasPrefix(filepath.Base(filepath.Dir(path)), "kongtrol-wg-")
}
