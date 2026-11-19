package daemonctl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// RecentConnections mirrors GET /api/v1/connections/recent (ADR-0054).
type RecentConnections struct {
	Supported   bool                       `json:"supported"`
	Enabled     bool                       `json:"enabled"`
	Max         int                        `json:"max"`
	TTLSeconds  int                        `json:"ttl_seconds"`
	Connections []monitor.RecentConnection `json:"connections"`
}

// RecentConnections returns up to limit recent destinations (0 = all).
func (c *Client) RecentConnections(limit int) (RecentConnections, error) {
	var out RecentConnections
	path := "/api/v1/connections/recent"
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}
	resp, err := c.do(http.MethodGet, path, 5*time.Second)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return out, APIError(resp)
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out)
	return out, err
}

// SetRecentConnections turns the recent destinations list on or off
// (PUT /api/v1/settings recent_connections_enabled; applies at once).
func (c *Client) SetRecentConnections(enabled bool) error {
	body, _ := json.Marshal(map[string]bool{"recent_connections_enabled": enabled})
	req, err := c.NewRequest(http.MethodPut, c.Base+"/api/v1/settings", bytes.NewReader(body))
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
