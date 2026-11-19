package daemonctl

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// UpdateState mirrors the API's UpdateStatus (/api/v1/updates).
type UpdateState struct {
	Enabled   bool       `json:"check_enabled"`
	Host      string     `json:"host"`
	Current   string     `json:"current"`
	Latest    string     `json:"latest,omitempty"`
	URL       string     `json:"url,omitempty"`
	Newer     bool       `json:"newer"`
	CheckedAt *time.Time `json:"checked_at,omitempty"`
}

// SetUpdateCheck turns the opt-in update notice on or off.
func (c *Client) SetUpdateCheck(enabled bool) (UpdateState, error) {
	var st UpdateState
	body, _ := json.Marshal(map[string]bool{"enabled": enabled})
	req, err := c.NewRequest(http.MethodPost, c.Base+"/api/v1/updates", bytes.NewReader(body))
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
