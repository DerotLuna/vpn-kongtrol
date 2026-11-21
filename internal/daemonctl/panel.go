package daemonctl

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DiagnosticsPath = "/settings?section=diagnostics"

// PanelURL returns the address that opens path ("/", "/profiles",
// "/tray", …) on the daemon's dashboard already signed in:
// it asks the daemon for a one-time nonce (valid 60 s, POST
// /api/v1/session/nonce with the control token) and appends it as
// "#k=<nonce>". The page trades the nonce for an HttpOnly session cookie
// (POST /api/v1/session) and removes it from the address bar. The control
// token itself never reaches the browser (ADR-0039).
//
// Every client that opens the dashboard (CLI `kongtrol dashboard`, the
// live view's "o", tray "Abrir panel", the tray flyout) must build its URL
// here. On error it still returns the plain URL: the page then shows how
// to open it from Kongtrol.
func (c *Client) PanelURL(path string) (string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	plain := c.Base + path
	resp, err := c.do(http.MethodPost, "/api/v1/session/nonce", 3*time.Second)
	if err != nil {
		return plain, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return plain, APIError(resp)
	}
	var body struct {
		Nonce string `json:"nonce"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body); err != nil {
		return plain, err
	}
	if body.Nonce == "" {
		return plain, io.ErrUnexpectedEOF
	}
	if i := strings.IndexByte(plain, '#'); i >= 0 {
		plain = plain[:i]
	}
	return plain + "#k=" + url.QueryEscape(body.Nonce), nil
}
