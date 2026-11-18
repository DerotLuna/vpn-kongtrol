package daemonctl

import (
	"encoding/json"
	"net/http"
	"time"
)

// Version asks the daemon for its build version (GET /api/v1/version),
// the same string `kongtrol version` prints.
func (c *Client) Version() (string, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/version", 3*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", APIError(resp)
	}
	var body struct {
		Version string `json:"version"`
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	return body.Version, err
}
