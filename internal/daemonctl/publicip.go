package daemonctl

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// PublicIPState is the daemon's opt-in public IP lookup state.
type PublicIPState struct {
	Enabled bool   `json:"public_ip_check_enabled"`
	Host    string `json:"public_ip_check_host"`
	// Check is the last (or just made) lookup; nil when there is none.
	Check *PublicIPCheck `json:"leak_check,omitempty"`
}

// PublicIPCheck mirrors the API's leak_check object.
type PublicIPCheck struct {
	State     string    `json:"state"` // clean | leak | error
	HasLeak   bool      `json:"has_leak"`
	PublicIP  string    `json:"public_ip,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// PublicIP reads the lookup state from /api/v1/security/status. No
// external request is made.
func (c *Client) PublicIP() (PublicIPState, error) {
	var st PublicIPState
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

// CheckPublicIP asks the daemon for one lookup now (the user's "Revisar").
// A disabled lookup answers a *RemoteError with key publicip.err.disabled.
func (c *Client) CheckPublicIP() (PublicIPState, error) {
	var st PublicIPState
	resp, err := c.do(http.MethodPost, "/api/v1/security/publicip/check", 15*time.Second)
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

// SetPublicIPCheck turns the lookup on or off; it returns the service host.
func (c *Client) SetPublicIPCheck(enabled bool) (PublicIPState, error) {
	var st PublicIPState
	body, _ := json.Marshal(map[string]bool{"enabled": enabled})
	req, err := c.NewRequest(http.MethodPost, c.Base+"/api/v1/security/publicip", bytes.NewReader(body))
	if err != nil {
		return st, err
	}
	resp, err := HTTPClient(10 * time.Second).Do(req)
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
