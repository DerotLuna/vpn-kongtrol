package daemonctl

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// AuditLogSettings are the audit log fields of GET /api/v1/settings: the
// values in use (a 0 in kongtrol.yaml comes back as the default, ADR-0064).
type AuditLogSettings struct {
	Path          string `json:"audit_log_path"`
	MaxSizeMB     int    `json:"audit_log_max_size_mb"`
	RetentionDays int    `json:"audit_log_retention_days"`
	MaxTotalMB    int    `json:"audit_log_max_total_mb"`
	Sign          bool   `json:"audit_log_sign"`
}

// AuditLogPatch changes only the fields that are set
// (PUT /api/v1/settings; the daemon validates and applies them at once).
type AuditLogPatch struct {
	MaxSizeMB     *int `json:"audit_log_max_size_mb,omitempty"`
	RetentionDays *int `json:"audit_log_retention_days,omitempty"`
	MaxTotalMB    *int `json:"audit_log_max_total_mb,omitempty"`
}

// AuditLog returns the audit log settings in use.
func (c *Client) AuditLog() (AuditLogSettings, error) {
	var out AuditLogSettings
	resp, err := c.do(http.MethodGet, "/api/v1/settings", 5*time.Second)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return out, APIError(resp)
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)
	return out, err
}

// SetAuditLog saves p. A refusal comes back as a *RemoteError with the
// translated message and "what to do" line.
func (c *Client) SetAuditLog(p AuditLogPatch) error {
	resp, err := c.sendJSON(http.MethodPut, "/api/v1/settings", p)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return APIError(resp)
	}
	return nil
}
