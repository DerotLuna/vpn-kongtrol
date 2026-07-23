package main

import (
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
