package main

import (
	"io"
	"net/http"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// The daemon REST client lives in internal/daemonctl so the tray drives the
// same daemon the same way. These wrappers bind it to this process's API
// token and keep the CLI's call sites and test seams unchanged.

// daemonAPIBase returns the address the running daemon's embedded API
// server binds to, per the loaded config. It's a fixed, well-known address
// (default 127.0.0.1:9741) — every kongtrol process reads the same config,
// so this is how any viewer finds a daemon that's already running.
func daemonAPIBase() string {
	return daemonctl.APIBase(cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port)
}

// daemonClient returns an API client for base using this process's token.
func daemonClient(base string) *daemonctl.Client { return daemonctl.New(base, apiToken) }

// probeDaemonAPI reports whether a kongtrol daemon's API server is reachable
// at base.
func probeDaemonAPI(base string) bool { return daemonClient(base).Probe() }

// daemonWSURL converts the daemon API's http(s) base address into the
// ws(s) URL for its live metrics feed.
func daemonWSURL(base string) string { return daemonctl.WSURL(base) }

// daemonAPIError reads a JSON error body and falls back to the raw body /
// status text if it doesn't parse.
func daemonAPIError(resp *http.Response) error { return daemonctl.APIError(resp) }

// daemonConnect asks the running daemon to (re)connect profile name. The
// daemon connects asynchronously (202 Accepted) — this only confirms the
// request was accepted.
func daemonConnect(base, name string) error {
	_, err := daemonRequestConnect(base, name)
	return err
}

// daemonRequestConnect is daemonConnect that also reports whether the
// daemon answered "already_connected" (200) instead of starting a new
// asynchronous connect (202).
func daemonRequestConnect(base, name string) (alreadyConnected bool, err error) {
	return daemonClient(base).Connect(name)
}

// daemonCancelConnect aborts a pending connect for profile name.
func daemonCancelConnect(base, name string) error { return daemonClient(base).CancelConnect(name) }

// daemonDisconnect disconnects profile name synchronously (bounded to 30s
// server-side), so a successful return means the tunnel is already down.
func daemonDisconnect(base, name string) error { return daemonClient(base).Disconnect(name) }

// daemonShutdown asks a running daemon to terminate gracefully via its
// REST API — see handleShutdown's doc comment for why this is preferred
// over an OS-level kill/signal.
func daemonShutdown(base string) error { return daemonClient(base).Shutdown() }

// daemonRequest builds a request authenticated with this process's token.
func daemonRequest(method, url string, body io.Reader) (*http.Request, error) {
	return daemonClient("").NewRequest(method, url, body)
}

// daemonReloadPolicy asks the running daemon to re-read kongtrol.yaml from
// disk and hot-swap its policy engine from it, for the "I hand-edited
// kongtrol.yaml outside the dashboard/CLI" case `kongtrol reload` exists for.
func daemonReloadPolicy(base string) error { return daemonClient(base).ReloadPolicy() }

// daemonGroupReloadResult mirrors POST /api/v1/groups/{name}/reload.
type daemonGroupReloadResult = daemonctl.GroupReloadResult

// daemonReloadGroup asks the running daemon to re-read kongtrol.yaml and
// restart-in-place every currently connected profile in group name.
func daemonReloadGroup(base, name string) (daemonGroupReloadResult, error) {
	return daemonClient(base).ReloadGroup(name)
}

// daemonTunnelReloadResult mirrors POST /api/v1/tunnels/{name}/reload.
type daemonTunnelReloadResult = daemonctl.TunnelReloadResult

// daemonReloadTunnel asks the running daemon to re-read kongtrol.yaml and
// restart-in-place profile name if it's currently connected.
func daemonReloadTunnel(base, name string) (daemonTunnelReloadResult, error) {
	return daemonClient(base).ReloadTunnel(name)
}

// daemonReconnect disconnects then reconnects profile name through the
// daemon's API — /connect alone doesn't restart an already-connected one.
func daemonReconnect(base, name string) error {
	if err := daemonDisconnect(base, name); err != nil {
		return err
	}
	return daemonConnect(base, name)
}

// daemonTunnelStatuses returns the running daemon's view of every tunnel's
// status, keyed by profile name. It comes from the daemon's collector
// snapshot, so it can lag a real state change by up to one collector poll.
func daemonTunnelStatuses(base string) (map[string]vpn.Status, error) {
	return daemonClient(base).Statuses()
}

// daemonTunnelStates is daemonTunnelStatuses with each tunnel's full
// metrics, including its last connect error.
func daemonTunnelStates(base string) (map[string]monitor.TunnelMetrics, error) {
	return daemonClient(base).TunnelStates()
}

// daemonActiveProfiles returns the names of every profile the running
// daemon currently reports as connected or connecting. Because the list
// comes from the collector snapshot it can lag a just-finished disconnect —
// callers that just disconnected profiles subtract those themselves (see
// daemonRemainingAfter).
func daemonActiveProfiles(base string) ([]string, error) {
	statuses, err := daemonTunnelStatusesFn(base)
	if err != nil {
		return nil, err
	}
	var names []string
	for name, st := range statuses {
		if st == vpn.StatusConnected || st == vpn.StatusConnecting {
			names = append(names, name)
		}
	}
	return names, nil
}

// Indirections over the daemon client calls used by `up`, `down` and
// `daemon`, so their daemon-vs-local decision logic can be tested without
// a real daemon.
var (
	daemonProbeFn          = probeDaemonAPI
	daemonConnectFn        = daemonRequestConnect
	daemonCancelConnectFn  = daemonCancelConnect
	daemonDisconnectFn     = daemonDisconnect
	daemonTunnelStatusesFn = daemonTunnelStatuses
	daemonTunnelStatesFn   = daemonTunnelStates
	daemonActiveProfilesFn = daemonActiveProfiles
)
