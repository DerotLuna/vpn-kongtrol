package daemonctl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// HelperStatus mirrors GET /api/v1/helper/status.
type HelperStatus struct {
	Installed     bool   `json:"installed"`
	Running       bool   `json:"running"`
	Available     bool   `json:"available"`
	Elevated      bool   `json:"elevated"`
	HelperVersion string `json:"helper_version"`
	CLIVersion    string `json:"cli_version"`
	NeedsInstall  bool   `json:"needs_install"`
}

// HelperInstallError is a failed POST /api/v1/helper/install: Message and
// Hint are already translated into the language the request asked for.
type HelperInstallError struct {
	Status  int
	Message string
	Hint    string
	Key     string
	Detail  string
}

func (e *HelperInstallError) Error() string {
	if e.Hint == "" {
		return e.Message
	}
	return e.Message + " " + e.Hint
}

// helperInstallTimeout covers the daemon's 3-minute prompt window.
const helperInstallTimeout = 4 * time.Minute

// HelperStatus asks the daemon whether the privileged helper answers.
func (c *Client) HelperStatus() (HelperStatus, error) {
	var st HelperStatus
	resp, err := c.do(http.MethodGet, "/api/v1/helper/status", 5*time.Second)
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

// InstallHelper asks the daemon to install the helper; the OS shows one
// permission prompt and this blocks until the user answers (up to a few
// minutes). A refused or failed install is a *HelperInstallError in lang.
func (c *Client) InstallHelper(lang i18n.Lang) (HelperStatus, error) {
	var st HelperStatus
	req, err := c.NewRequest(http.MethodPost, c.Base+"/api/v1/helper/install", nil)
	if err != nil {
		return st, err
	}
	req.Header.Set("X-Kongtrol-Lang", string(lang))
	resp, err := (&http.Client{Timeout: helperInstallTimeout}).Do(req)
	if err != nil {
		return st, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		var e struct {
			Error    string `json:"error"`
			Hint     string `json:"hint"`
			ErrorKey string `json:"error_key"`
			Detail   string `json:"detail"`
		}
		if json.Unmarshal(body, &e) != nil || e.Error == "" {
			return st, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		return st, &HelperInstallError{Status: resp.StatusCode, Message: e.Error, Hint: e.Hint, Key: e.ErrorKey, Detail: e.Detail}
	}
	err = json.NewDecoder(resp.Body).Decode(&st)
	return st, err
}
