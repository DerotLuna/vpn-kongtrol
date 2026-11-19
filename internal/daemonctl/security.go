package daemonctl

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// SecurityState is the kill switch and DNS guard part of
// GET /api/v1/security/status.
type SecurityState struct {
	KillSwitchEnabled bool                       `json:"kill_switch_enabled"`
	KillSwitchStatus  *security.KillSwitchStatus `json:"kill_switch_status,omitempty"`
	DNSGuardEnabled   bool                       `json:"dns_guard_enabled"`
	// DNSGuardState is disabled | idle | active.
	DNSGuardState string `json:"dns_guard_state"`
	// DNSGuardServers are the DNS servers the guard applies now.
	DNSGuardServers []string `json:"dns_guard_servers"`
	// KillSwitchError is the last apply failure (clients render ErrorKey in
	// their own language); nil when the last apply worked.
	KillSwitchError *KillSwitchError `json:"kill_switch_error,omitempty"`
}

// KillSwitchError is GET /api/v1/security/status kill_switch_error.
type KillSwitchError struct {
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	ErrorKey string `json:"error_key"`
	Detail   string `json:"detail,omitempty"`
}

// Blocked reads the prefixes the kill switch blocks now
// (GET /api/v1/security/blocked).
func (c *Client) Blocked() ([]string, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/security/blocked", 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, APIError(resp)
	}
	var body struct {
		Prefixes []string `json:"prefixes"`
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	return body.Prefixes, err
}

// Security reads the kill switch and DNS guard state.
func (c *Client) Security() (SecurityState, error) {
	var st SecurityState
	resp, err := c.do(http.MethodGet, "/api/v1/security/status", 3*time.Second)
	if err != nil {
		return st, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return st, APIError(resp)
	}
	err = json.NewDecoder(resp.Body).Decode(&st)
	return st, err
}

// SetKillSwitch turns the kill switch on or off (applied at once).
func (c *Client) SetKillSwitch(enabled bool) error {
	return c.postEnabled("/api/v1/security/killswitch", enabled)
}

// SetDNSGuard turns the DNS guard on or off (applied at once).
func (c *Client) SetDNSGuard(enabled bool) error {
	return c.postEnabled("/api/v1/security/dnsguard", enabled)
}

func (c *Client) postEnabled(path string, enabled bool) error {
	body, _ := json.Marshal(map[string]bool{"enabled": enabled})
	req, err := c.NewRequest(http.MethodPost, c.Base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := HTTPClient(10 * time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return APIError(resp)
	}
	return nil
}
