package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
)

// importReq is the POST /api/v1/vpns/import body. The global middleware
// already requires application/json and caps the body at 1 MiB.
type importReq struct {
	Name          string `json:"name"`           // file name without extension
	Type          string `json:"type"`           // wireguard | openvpn
	ContentBase64 string `json:"content_base64"` // the .conf / .ovpn file
	Overwrite     bool   `json:"overwrite"`      // replace an existing file of that name
}

// SetImportDir sets where imported files go (default ~/.kongtrol/configs).
func (s *Server) SetImportDir(dir string) { s.importDir = dir }

func (s *Server) configsDir() (string, error) {
	if s.importDir != "" {
		return s.importDir, nil
	}
	return profileimport.DefaultDir()
}

// POST /api/v1/vpns/import — stores a WireGuard .conf or OpenVPN .ovpn the
// user picked in ~/.kongtrol/configs/<name>.<ext> (0600) and returns its
// path for the profile form's "config" field. Validation and storage are
// internal/profileimport, shared with `kongtrol profile import`: the
// helper's own checks (no hook scripts, no directives that run programs or
// write files, an imported .ovpn self-contained); never overwrites unless
// "overwrite": true (ADR-0042).
func (s *Server) handleImportVPNFile(w http.ResponseWriter, r *http.Request) {
	var req importReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, errInvalidJSON)
		return
	}
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.ContentBase64))
	if err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.import_content"})
		return
	}
	dir, err := s.configsDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	res, err := profileimport.Import(dir, profileimport.Request{
		Name: req.Name, Type: req.Type, Content: content, Overwrite: req.Overwrite,
	})
	if err != nil {
		writeLocalizedError(w, r, importStatus(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status": "imported", "path": res.Path, "type": res.Type, "overwritten": res.Overwritten,
	})
}

// importStatus: 409 for a name already taken, 413 for a file too big, 500
// when writing failed, 400 for any other refusal.
func importStatus(err error) int {
	var ve *config.ValidationError
	if !errors.As(err, &ve) {
		return http.StatusInternalServerError
	}
	switch ve.Key {
	case "api.err.import_exists", "api.err.import_not_file":
		return http.StatusConflict
	case "api.err.import_too_big":
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}
