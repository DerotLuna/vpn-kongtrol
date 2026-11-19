package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
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
	// Path instead of content: a file on this machine (absolute). One only
	// root can read is refused with config.err.file_root_only unless
	// Elevate is set: then the daemon asks for permission with the OS
	// dialog and reads it (ADR-0058).
	Path    string `json:"path,omitempty"`
	Elevate bool   `json:"elevate,omitempty"`
}

// ErrKeyElevatedReadFailed: the permission prompt was dismissed or the
// elevated read failed (the "what to do" line offers the manual copy).
const ErrKeyElevatedReadFailed = "api.err.import_elevated_failed"

// importReadFile reads a "path" import (a variable so tests can answer
// "permission denied" on any OS).
var importReadFile = os.ReadFile

// SetImportDir sets where imported files go (default ~/.kongtrol/configs).
func (s *Server) SetImportDir(dir string) { s.importDir = dir }

// SetProtectedReader sets how a root-only file is read with one OS
// permission prompt (pkexec on Linux). Without it, "elevate" answers
// config.err.file_root_only (its "what to do" line explains the copy).
func (s *Server) SetProtectedReader(fn func(ctx context.Context, path string) ([]byte, error)) {
	s.readProtected = fn
}

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
// "overwrite": true (ADR-0042). With "path" it reads that file instead,
// with permission for a root-only one when "elevate" is true (ADR-0058).
func (s *Server) handleImportVPNFile(w http.ResponseWriter, r *http.Request) {
	var req importReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, errInvalidJSON)
		return
	}
	var content []byte
	var err error
	if strings.TrimSpace(req.Path) != "" {
		if content, err = s.readImportPath(r.Context(), &req); err != nil {
			writeLocalizedError(w, r, importStatus(err), err)
			return
		}
	} else if content, err = base64.StdEncoding.DecodeString(strings.TrimSpace(req.ContentBase64)); err != nil {
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

// readImportPath reads req.Path (absolute, .conf / .ovpn) and fills Type
// and Name from it when empty. A root-only file needs Elevate.
func (s *Server) readImportPath(ctx context.Context, req *importReq) ([]byte, error) {
	path := strings.TrimSpace(req.Path)
	refuse := &config.ValidationError{Key: profileimport.ErrKeyProtectedPath, Args: []any{path}}
	typ := profileimport.TypeForFile(path)
	if !filepath.IsAbs(path) || typ == "" {
		return nil, refuse
	}
	if req.Type == "" {
		req.Type = typ
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = profileimport.CleanName(path)
	}
	fi, err := os.Stat(path)
	if err == nil {
		if !fi.Mode().IsRegular() {
			return nil, refuse
		}
		if fi.Size() > profileimport.MaxSize {
			return nil, profileimport.TooBig()
		}
		var b []byte
		if b, err = importReadFile(path); err == nil {
			return b, nil
		}
	}
	if !config.IsPermission(err) {
		return nil, refuse
	}
	if !req.Elevate || s.readProtected == nil {
		return nil, &config.ValidationError{Key: config.ErrKeyFileRootOnly, Args: []any{path}}
	}
	b, err := s.readProtected(ctx, path)
	if err != nil {
		var ve *config.ValidationError
		if errors.As(err, &ve) {
			return nil, err
		}
		return nil, &config.ValidationError{Key: ErrKeyElevatedReadFailed, Args: []any{path}}
	}
	return b, nil
}

// importStatus: 409 for a name already taken, 413 for a file too big, 403
// for a root-only file, 500 when writing failed, 400 for any other refusal.
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
	case config.ErrKeyFileRootOnly, ErrKeyElevatedReadFailed:
		return http.StatusForbidden
	}
	return http.StatusBadRequest
}
