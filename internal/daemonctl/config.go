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
		"disabled":       rule.Disabled,
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

// UpdateVPN applies a partial profile edit through the daemon.
func (c *Client) UpdateVPN(name string, p configedit.VPNProfile) error {
	return c.updateConfig("/api/v1/vpns/"+url.PathEscape(name), p)
}

// UpdatePolicy applies a partial policy edit through the daemon.
func (c *Client) UpdatePolicy(name string, p configedit.PolicyPatch) error {
	return c.updateConfig("/api/v1/policies/"+url.PathEscape(name), p)
}

func (c *Client) updateConfig(path string, p any) error {
	resp, err := c.sendJSON(http.MethodPut, path, p)
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

// DefaultEgress checks compatibility, or applies the supplied selection.
func (c *Client) DefaultEgress(name string, enabled *bool) (config.DefaultEgressStatus, error) {
	path := "/api/v1/vpns/" + url.PathEscape(name) + "/default-egress"
	var result config.DefaultEgressStatus
	var resp *http.Response
	var err error
	if enabled == nil {
		resp, err = c.sendJSON(http.MethodGet, path, nil)
	} else {
		resp, err = c.sendJSON(http.MethodPut, path, map[string]bool{"enabled": *enabled})
	}
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return result, APIError(resp)
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// MovePolicies points every rule of profile from to profile to in one
// validated save (POST /api/v1/policies/move, ADR-0104) and returns the
// moved rule names.
func (c *Client) MovePolicies(from, to string) ([]string, error) {
	resp, err := c.sendJSON(http.MethodPost, "/api/v1/policies/move", map[string]string{"from": from, "to": to})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, APIError(resp)
	}
	var out struct {
		Moved []string `json:"moved"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.Moved, nil
}
