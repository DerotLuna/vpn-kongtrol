// Package daemonctl is the client side of the kongtrol daemon: the REST
// client every viewer (CLI, tray) uses to drive it, and the logic that
// spawns a detached `kongtrol daemon` when none answers. Both binaries
// share it so they find, start and talk to the same single daemon.
package daemonctl

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// TokenHeader carries the local API control token on every request.
const TokenHeader = "X-Kongtrol-Token"

// APIBase returns the http base URL of the daemon API for a dashboard
// bind/port pair. A wildcard bind (0.0.0.0 / ::) is a listen address, not
// a dial target — Windows refuses to connect to it — so it maps to
// loopback.
func APIBase(bind string, port int) string {
	host := bind
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}

// WSURL converts the API's http(s) base address into the ws(s) URL of its
// live metrics feed.
func WSURL(base string) string {
	url := strings.Replace(base, "http://", "ws://", 1)
	url = strings.Replace(url, "https://", "wss://", 1)
	return url + "/api/v1/ws/metrics"
}

// Client talks to one daemon's REST API.
type Client struct {
	Base  string
	Token string
	// TokenSource, when set, is read on every request and wins over Token:
	// the daemon writes a new ~/.kongtrol/api-token on every start
	// (ADR-0062), so a long-lived client (tray, live view) must not keep
	// the one it read at launch. Token is the fallback when it fails.
	TokenSource func() (string, error)
	// Lang ("es", "en"; optional) asks the daemon to word its translated
	// errors in the client's language (X-Kongtrol-Lang).
	Lang string
}

// New returns a client for the daemon at base, authenticated with token.
func New(base, token string) *Client { return &Client{Base: base, Token: token} }

// CurrentToken is the token to send now (TokenSource, else Token).
func (c *Client) CurrentToken() string {
	if c.TokenSource != nil {
		if t, err := c.TokenSource(); err == nil && t != "" {
			return t
		}
	}
	return c.Token
}

// NewRequest builds an authenticated request; url is absolute. Send it
// with HTTPClient, which checks who owns the listener first.
func (c *Client) NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set(TokenHeader, c.CurrentToken())
	if c.Lang != "" {
		req.Header.Set("X-Kongtrol-Lang", c.Lang)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// APIError reads a JSON error body of the shape {"error": "..."} and falls
// back to the raw body / status text if it doesn't parse.

// do sends a body-less request for path with timeout; the caller checks
// the status and closes the body.
func (c *Client) do(method, path string, timeout time.Duration) (*http.Response, error) {
	req, err := c.NewRequest(method, c.Base+path, nil)
	if err != nil {
		return nil, err
	}
	return HTTPClient(timeout).Do(req)
}

// post sends a body-less POST and discards a successful response.
func (c *Client) post(path string, timeout time.Duration) error {
	resp, err := c.do(http.MethodPost, path, timeout)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return APIError(resp)
	}
	return nil
}

// Probe reports whether a kongtrol daemon's API answers at Base.
func (c *Client) Probe() bool {
	resp, err := c.do(http.MethodGet, "/api/v1/tunnels", 800*time.Millisecond)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

// Connect asks the daemon to connect profile name. The daemon connects
// asynchronously (202); alreadyConnected is true when it answered 200
// "already_connected" instead. The outcome of an accepted connect shows up
// in Tunnels (status, and LastError when it fails).
func (c *Client) Connect(name string) (alreadyConnected bool, err error) {
	resp, err := c.do(http.MethodPost, "/api/v1/tunnels/"+name+"/connect", 10*time.Second)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return false, APIError(resp)
	}
	return resp.StatusCode == http.StatusOK, nil
}

// CancelConnect aborts a pending connect for profile name.
func (c *Client) CancelConnect(name string) error {
	return c.post("/api/v1/tunnels/"+name+"/cancel_connect", 35*time.Second)
}

// Disconnect disconnects profile name; the daemon does it synchronously
// (bounded to 30s server-side), so success means the tunnel is down.
//
// A provider app without a CLI answers 202 {"status":"waiting_user"}: the
// user must disconnect in the app. Disconnect then returns an error
// wrapping vpn.ErrManualDisconnect; callers poll TunnelStates (the
// tunnel's Attach phase is waiting_disconnect) until it drops.
func (c *Client) Disconnect(name string) error {
	resp, err := c.do(http.MethodPost, "/api/v1/tunnels/"+name+"/disconnect", 35*time.Second)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return APIError(resp)
	}
	if resp.StatusCode == http.StatusAccepted {
		var body struct {
			Status string `json:"status"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body) == nil && body.Status == "waiting_user" {
			return fmt.Errorf("%s: %w", name, vpn.ErrManualDisconnect)
		}
	}
	return nil
}

// Shutdown asks the daemon to disconnect everything and exit gracefully.
func (c *Client) Shutdown() error {
	return c.post("/api/v1/shutdown", 5*time.Second)
}

// ReloadPolicy asks the daemon to re-read kongtrol.yaml and hot-swap its
// policy engine.
func (c *Client) ReloadPolicy() error {
	_, err := c.ReloadConfig()
	return err
}

// ConfigReloadResult mirrors POST /api/v1/policies/reload (ADR-0053).
type ConfigReloadResult struct {
	// Live is false for an older daemon that does not apply profiles
	// without a restart (no restart_required field in its answer).
	Live bool `json:"-"`
	// RestartRequired: some profile could not be applied live.
	RestartRequired *bool `json:"restart_required"`
	// ReconnectRequired: connected profiles whose settings changed; they
	// keep the old ones until reconnected.
	ReconnectRequired []string `json:"reconnect_required"`
}

// ReloadConfig re-reads kongtrol.yaml in the daemon and says which
// profiles need a reconnect or a restart.
func (c *Client) ReloadConfig() (ConfigReloadResult, error) {
	var out ConfigReloadResult
	resp, err := c.do(http.MethodPost, "/api/v1/policies/reload", 10*time.Second)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return out, APIError(resp)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return ConfigReloadResult{}, nil // an answer without a body: older daemon
	}
	out.Live = out.RestartRequired != nil
	return out, nil
}

// Tunnels returns the daemon's per-tunnel metrics, including each one's
// last connect error.
func (c *Client) Tunnels() ([]monitor.TunnelMetrics, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/tunnels", 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, APIError(resp)
	}
	var tunnels []monitor.TunnelMetrics
	if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
		return nil, err
	}
	return tunnels, nil
}

// TunnelStates returns Tunnels keyed by profile name, statuses normalized.
func (c *Client) TunnelStates() (map[string]monitor.TunnelMetrics, error) {
	tunnels, err := c.Tunnels()
	if err != nil {
		return nil, err
	}
	out := make(map[string]monitor.TunnelMetrics, len(tunnels))
	for _, t := range tunnels {
		t.Status = t.Status.Normalize()
		out[t.Name] = t
	}
	return out, nil
}

// Statuses returns every tunnel's normalized status keyed by profile name.
func (c *Client) Statuses() (map[string]vpn.Status, error) {
	states, err := c.TunnelStates()
	if err != nil {
		return nil, err
	}
	out := make(map[string]vpn.Status, len(states))
	for name, t := range states {
		out[name] = t.Status
	}
	return out, nil
}

// GroupReloadResult mirrors POST /api/v1/groups/{name}/reload, for both the
// 202 "restarting" shape and the 409 "restart_required" shape.
type GroupReloadResult struct {
	Status          string   `json:"status"`
	Group           string   `json:"group"`
	Restarted       []string `json:"restarted"`
	Skipped         []string `json:"skipped"`
	MissingProfiles []string `json:"missing_profiles"`
	RestartRequired bool     `json:"-"`
}

// ReloadGroup asks the daemon to re-read kongtrol.yaml and restart in place
// every connected profile of group name.
func (c *Client) ReloadGroup(name string) (GroupReloadResult, error) {
	var result GroupReloadResult
	restart, err := c.reload("/api/v1/groups/"+name+"/reload", &result)
	result.RestartRequired = restart
	return result, err
}

// TunnelReloadResult mirrors POST /api/v1/tunnels/{name}/reload.
type TunnelReloadResult struct {
	Status          string   `json:"status"`
	Tunnel          string   `json:"tunnel"`
	Restarted       []string `json:"restarted"`
	Skipped         []string `json:"skipped"`
	RestartRequired bool     `json:"-"`
}

// ReloadTunnel asks the daemon to re-read kongtrol.yaml and restart profile
// name in place if it is connected.
func (c *Client) ReloadTunnel(name string) (TunnelReloadResult, error) {
	var result TunnelReloadResult
	restart, err := c.reload("/api/v1/tunnels/"+name+"/reload", &result)
	result.RestartRequired = restart
	return result, err
}

// reload posts to a reload endpoint and decodes its body into out.
// restartRequired is true for a decodable 409 answer.
func (c *Client) reload(path string, out any) (restartRequired bool, err error) {
	resp, err := c.do(http.MethodPost, path, 35*time.Second)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		if json.NewDecoder(resp.Body).Decode(out) == nil {
			return true, nil
		}
	}
	if resp.StatusCode >= 300 {
		return false, APIError(resp)
	}
	_ = json.NewDecoder(resp.Body).Decode(out)
	return false, nil
}

// RemoteError is a non-2xx daemon answer. Key is the i18n key of a
// one-line guidance message when the daemon tagged the error with one
// (body "error_key"); clients render it in their own language.
type RemoteError struct {
	Status  string
	Message string
	Key     string
	// Hint is the "what to do" line of a translated validation error
	// (body "hint"), already in the language the request asked for.
	Hint string
}

// UserText is the message plus its "what to do" line for a translated
// validation error (Hint set); ok is false for any other answer.
func (e *RemoteError) UserText() (text string, ok bool) {
	if e.Hint == "" || e.Message == "" {
		return "", false
	}
	return e.Message + "\n    " + e.Hint, true
}

func (e *RemoteError) Error() string {
	if e.Message == "" {
		return e.Status
	}
	return e.Status + ": " + e.Message
}

// ErrorKey returns the guidance i18n key ("" when none).
func (e *RemoteError) ErrorKey() string { return e.Key }

// APIError builds a *RemoteError from a non-2xx response.
func APIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	e := &RemoteError{Status: resp.Status, Message: string(body)}
	var j struct {
		Error    string `json:"error"`
		ErrorKey string `json:"error_key"`
		Detail   string `json:"detail"`
		Hint     string `json:"hint"`
	}
	if json.Unmarshal(body, &j) == nil && j.ErrorKey != "" {
		e.Key = j.ErrorKey
		e.Message = j.Error
		e.Hint = j.Hint
		if j.Detail != "" {
			e.Message = j.Detail
		}
	}
	return e
}

// Events returns the daemon notices newer than after (keys + args, render
// them with internal/i18n) and the latest event number to pass next time.
// after < 0 returns no backlog, only the latest number.
func (c *Client) Events(after int64) ([]monitor.Event, int64, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/events?after="+strconv.FormatInt(after, 10), 3*time.Second)
	if err != nil {
		return nil, after, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, after, APIError(resp)
	}
	var body struct {
		Events []monitor.Event `json:"events"`
		Seq    int64           `json:"seq"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, after, err
	}
	return body.Events, body.Seq, nil
}

// KillSwitch returns the daemon's split-tunnel kill switch state (nil when
// the daemon is older and does not report it).
func (c *Client) KillSwitch() (*security.KillSwitchStatus, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/security/status", 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, APIError(resp)
	}
	var body struct {
		Status *security.KillSwitchStatus `json:"kill_switch_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Status, nil
}
