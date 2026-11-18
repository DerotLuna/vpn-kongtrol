package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// importNameRE: the stored file name (before the extension). Letters,
// digits, dot, dash, underscore; starts with a letter or digit; ≤ 64.
// No separators, so no path traversal; no leading dot (hidden files).
var importNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// windowsReserved are device names Windows maps to devices in any folder.
var windowsReserved = map[string]bool{"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true}

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
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kongtrol", "configs"), nil
}

// POST /api/v1/vpns/import — stores a WireGuard .conf or OpenVPN .ovpn the
// user picked in ~/.kongtrol/configs/<name>.<ext> (0600) and returns its
// path for the profile form's "config" field. The content is validated
// with the same checks the privileged helper applies before running it
// (privops.ValidateWireGuardConfig / SanitizeOpenVPNConfig): no hook
// scripts, no directives that run programs or write files, and — for an
// imported .ovpn — no references to other files (they must be inline, so
// an import never makes Kongtrol read some other path). Never overwrites
// unless "overwrite": true (ADR-0042).
func (s *Server) handleImportVPNFile(w http.ResponseWriter, r *http.Request) {
	var req importReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, errInvalidJSON)
		return
	}
	name := strings.TrimSpace(req.Name)
	if !importNameRE.MatchString(name) || strings.Contains(name, "..") || windowsReserved[strings.ToLower(strings.SplitN(name, ".", 2)[0])] {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.import_name", Args: []any{name}})
		return
	}
	var ext string
	switch strings.ToLower(strings.TrimSpace(req.Type)) {
	case "wireguard":
		ext = ".conf"
	case "openvpn":
		ext = ".ovpn"
	default:
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.import_type", Args: []any{req.Type}})
		return
	}
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.ContentBase64))
	if err != nil || len(content) == 0 {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.import_content"})
		return
	}
	if len(content) > privops.MaxConfigSize {
		writeLocalizedError(w, r, http.StatusRequestEntityTooLarge, &config.ValidationError{Key: "api.err.import_too_big", Args: []any{privops.MaxConfigSize >> 10}})
		return
	}
	if err := validateImport(ext, content); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.import_invalid", Args: []any{err.Error()}})
		return
	}

	dir, err := s.configsDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	path := filepath.Join(dir, name+ext)
	overwritten, err := storeImport(path, content, req.Overwrite)
	switch {
	case errors.Is(err, fs.ErrExist):
		writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "api.err.import_exists", Args: []any{name + ext}})
		return
	case errors.Is(err, errNotRegular):
		writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "api.err.import_not_file", Args: []any{name + ext}})
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status": "imported", "path": path, "type": strings.ToLower(req.Type), "overwritten": overwritten,
	})
}

// validateImport runs the helper's own checks. An .ovpn must be
// self-contained: every file reference is refused (no reader).
func validateImport(ext string, content []byte) error {
	if ext == ".conf" {
		return privops.ValidateWireGuardConfig(content)
	}
	_, err := privops.SanitizeOpenVPNConfig(content, func(p string) ([]byte, error) {
		return nil, fmt.Errorf("it references the file %q: put its content inline (<ca>…</ca>, <cert>, <key>…)", p)
	})
	return err
}

var errNotRegular = errors.New("not a regular file")

// storeImport writes content to path with 0600. Without overwrite it
// creates the file exclusively (O_EXCL: never replaces, even in a race);
// with overwrite it replaces a regular file atomically, never a link or a
// folder.
func storeImport(path string, content []byte, overwrite bool) (bool, error) {
	fi, err := os.Lstat(path)
	exists := err == nil
	if exists && !fi.Mode().IsRegular() {
		return false, errNotRegular
	}
	if !overwrite {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return false, err // fs.ErrExist when it is there
		}
		_, werr := f.Write(content)
		cerr := f.Close()
		if err := errors.Join(werr, cerr); err != nil {
			_ = os.Remove(path)
			return false, err
		}
		return false, nil
	}
	return exists, config.WriteFileAtomic(path, content, 0o600)
}
