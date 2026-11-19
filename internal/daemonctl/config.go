package daemonctl

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
)

// Config edits through the daemon (the dashboard's endpoints). Without a
// daemon the CLI applies the same internal/configedit functions to
// kongtrol.yaml itself.

// sendJSON sends body as JSON to path and returns the response; the caller
// checks the status and closes the body.
func (c *Client) sendJSON(method, path string, body any) (*http.Response, error) {
	var rd *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(data)
	}
	var req *http.Request
	var err error
	if rd != nil {
		req, err = c.NewRequest(method, c.Base+path, rd)
	} else {
		req, err = c.NewRequest(method, c.Base+path, nil)
	}
	if err != nil {
		return nil, err
	}
	return HTTPClient(15 * time.Second).Do(req)
}

// CreatePolicy adds rule (POST /api/v1/policies). A refusal comes back as a
// *RemoteError with the translated message and "what to do" line.
func (c *Client) CreatePolicy(rule config.PolicyRule) error {
	body := map[string]any{
		"name":           rule.Name,
		"via":            rule.Via,
		"on_tunnel_down": rule.OnTunnelDown,
		"match": map[string][]string{
			"domains":   nonNil(rule.Match.Domains),
			"ip_ranges": nonNil(rule.Match.IPRanges),
			"apps":      nonNil(rule.Match.Apps),
		},
	}
	resp, err := c.sendJSON(http.MethodPost, "/api/v1/policies", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return APIError(resp)
	}
	return nil
}

// DeletePolicy removes the policy named name (DELETE /api/v1/policies/{name}).
func (c *Client) DeletePolicy(name string) error {
	resp, err := c.sendJSON(http.MethodDelete, "/api/v1/policies/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return APIError(resp)
	}
	return nil
}

// CreateVPN creates a profile (POST /api/v1/vpns). A password in p goes
// over the loopback API to the daemon, which stores it in the keychain,
// as the dashboard does. restartRequired is the daemon's flag: false when
// it applied the profile live (ADR-0053), true in setup mode or for an
// older daemon.
func (c *Client) CreateVPN(p configedit.VPNProfile) (restartRequired bool, err error) {
	resp, err := c.sendJSON(http.MethodPost, "/api/v1/vpns", p)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return false, APIError(resp)
	}
	var out struct {
		RestartRequired bool `json:"restart_required"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.RestartRequired, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
