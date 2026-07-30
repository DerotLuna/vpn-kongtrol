package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// daemonAPIBase returns the address the running `kongtrol up` daemon's
// embedded API server binds to, per the loaded config. It's a fixed,
// well-known address (default 127.0.0.1:9741) — every kongtrol process reads
// the same config, so this is how a `status --watch` viewer finds a daemon
// that's already running in another process.
func daemonAPIBase() string {
	return "http://" + net.JoinHostPort(cfg.Monitor.Dashboard.Bind, strconv.Itoa(cfg.Monitor.Dashboard.Port))
}

// probeDaemonAPI reports whether a kongtrol daemon's API server is reachable
// at base. Used by `status --watch` to decide whether connect/disconnect
// actions can be safely proxied to the real daemon instead of being run
// in-process (which would spawn a second, uncoordinated adapter instance and
// risk a duplicate tunnel — see the `up_tui.go` daemonMode comment).
func probeDaemonAPI(base string) bool {
	client := http.Client{Timeout: 800 * time.Millisecond}
	req, err := daemonRequest(http.MethodGet, base+"/api/v1/tunnels", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

// daemonWSURL converts the daemon API's http(s) base address into the
// ws(s) URL for its live metrics feed.
func daemonWSURL(base string) string {
	url := strings.Replace(base, "http://", "ws://", 1)
	url = strings.Replace(url, "https://", "wss://", 1)
	return url + "/api/v1/ws/metrics"
}

// daemonAPIError reads a JSON error body of the shape {"error": "..."} and
// falls back to the raw body / status text if it doesn't parse.
func daemonAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if len(body) == 0 {
		return fmt.Errorf("%s", resp.Status)
	}
	return fmt.Errorf("%s: %s", resp.Status, string(body))
}

// daemonConnect asks the running daemon to (re)connect profile name via its
// REST API. The daemon's handler runs the connect asynchronously (202
// Accepted) — this call only confirms the request was accepted; the tunnel
// table picks up the resulting state change on its own next poll, since
// adapter.Status() reflects real OS-level tunnel state regardless of which
// process initiated the connection.
func daemonConnect(base, name string) error {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := daemonRequest(http.MethodPost, base+"/api/v1/tunnels/"+name+"/connect", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return daemonAPIError(resp)
	}
	return nil
}

// daemonDisconnect asks the running daemon to disconnect profile name via
// its REST API. Unlike connect, the daemon's handler disconnects
// synchronously (bounded to 30s server-side), so a successful return means
// the tunnel is already down.
func daemonDisconnect(base, name string) error {
	client := http.Client{Timeout: 35 * time.Second}
	req, err := daemonRequest(http.MethodPost, base+"/api/v1/tunnels/"+name+"/disconnect", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return daemonAPIError(resp)
	}
	return nil
}

// daemonShutdown asks a running daemon to terminate gracefully via its
// REST API — see handleShutdown's doc comment for why this is preferred
// over an OS-level kill/signal.
func daemonShutdown(base string) error {
	client := http.Client{Timeout: 5 * time.Second}
	req, err := daemonRequest(http.MethodPost, base+"/api/v1/shutdown", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return daemonAPIError(resp)
	}
	return nil
}

func daemonRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Kongtrol-Token", apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// daemonReloadPolicy asks the running daemon to re-read kongtrol.yaml from
// disk and hot-swap its policy engine from it — the same hot-swap
// saveRuntimeConfig performs after a dashboard CRUD write, but sourced from
// the file on disk instead of an in-memory mutation, for the "I hand-edited
// kongtrol.yaml outside the dashboard/CLI" case `kongtrol reload` exists for.
func daemonReloadPolicy(base string) error {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := daemonRequest(http.MethodPost, base+"/api/v1/policies/reload", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return daemonAPIError(resp)
	}
	return nil
}

// daemonGroupNames returns the names of every group defined in the running
// daemon's currently loaded config. Used by `kongtrol reload` with no
// --group flag to restart every group in place.
func daemonGroupNames(base string) ([]string, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := daemonRequest(http.MethodGet, base+"/api/v1/groups", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, daemonAPIError(resp)
	}
	var groups []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	return names, nil
}

// daemonGroupReloadResult mirrors the JSON body returned by
// POST /api/v1/groups/{name}/reload, for both the 202 "restarting" success
// shape and the 409 "restart_required" shape (a brand-new profile added to
// the group by the hand edit that the running daemon has no registered
// adapter for — see handleReloadGroup's doc comment in internal/api).
type daemonGroupReloadResult struct {
	Status          string   `json:"status"`
	Group           string   `json:"group"`
	Restarted       []string `json:"restarted"`
	Skipped         []string `json:"skipped"`
	MissingProfiles []string `json:"missing_profiles"`
	RestartRequired bool     `json:"-"`
}

// daemonReloadGroup asks the running daemon to re-read kongtrol.yaml and
// restart-in-place every currently connected profile in group name.
func daemonReloadGroup(base, name string) (daemonGroupReloadResult, error) {
	client := http.Client{Timeout: 35 * time.Second}
	req, err := daemonRequest(http.MethodPost, base+"/api/v1/groups/"+name+"/reload", nil)
	if err != nil {
		return daemonGroupReloadResult{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return daemonGroupReloadResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		var result daemonGroupReloadResult
		if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr == nil {
			result.RestartRequired = true
			return result, nil
		}
	}
	if resp.StatusCode >= 300 {
		return daemonGroupReloadResult{}, daemonAPIError(resp)
	}
	var result daemonGroupReloadResult
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// daemonTunnelReloadResult mirrors the JSON body returned by
// POST /api/v1/tunnels/{name}/reload — the single-profile counterpart to
// daemonGroupReloadResult, for restarting one tunnel in place instead of a
// whole group.
type daemonTunnelReloadResult struct {
	Status          string   `json:"status"`
	Tunnel          string   `json:"tunnel"`
	Restarted       []string `json:"restarted"`
	Skipped         []string `json:"skipped"`
	RestartRequired bool     `json:"-"`
}

// daemonReloadTunnel asks the running daemon to re-read kongtrol.yaml and
// restart-in-place profile name if it's currently connected — a hand edit
// scoped to one tunnel doesn't need its whole group cycled.
func daemonReloadTunnel(base, name string) (daemonTunnelReloadResult, error) {
	client := http.Client{Timeout: 35 * time.Second}
	req, err := daemonRequest(http.MethodPost, base+"/api/v1/tunnels/"+name+"/reload", nil)
	if err != nil {
		return daemonTunnelReloadResult{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return daemonTunnelReloadResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		var result daemonTunnelReloadResult
		if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr == nil {
			result.RestartRequired = true
			return result, nil
		}
	}
	if resp.StatusCode >= 300 {
		return daemonTunnelReloadResult{}, daemonAPIError(resp)
	}
	var result daemonTunnelReloadResult
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// daemonReconnect disconnects then reconnects profile name through the
// daemon's API — there's no dedicated reconnect endpoint since /connect
// already reconnects a non-connected tunnel, but for an already-connected
// one we need the explicit disconnect first to match "reconnect" semantics.
func daemonReconnect(base, name string) error {
	if err := daemonDisconnect(base, name); err != nil {
		return err
	}
	return daemonConnect(base, name)
}
