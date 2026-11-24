package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
)

// importReq is the POST /api/v1/vpns/import body. The global middleware
// already requires application/json and caps this route's body at
// importBodyLimit.
type importReq struct {
	Name          string `json:"name"`           // file name without extension
	Type          string `json:"type"`           // wireguard | openvpn
	ContentBase64 string `json:"content_base64"` // the .conf / .ovpn file
	Overwrite     bool   `json:"overwrite"`      // replace an existing file of that name
	// Path instead of content: a file on this machine (absolute). One only
	// root can read is refused with config.err.file_root_only unless
	// Elevate is set: then the daemon asks for permission with the OS
	// dialog and reads it (ADR-0058). The files an .ovpn names are then
	// read from its folder (ADR-0105).
	Path    string `json:"path,omitempty"`
	Elevate bool   `json:"elevate,omitempty"`
	// Files are certificates and keys the user added by hand (dropped with
	// the .ovpn, or "Agregar <name>"), matched to the .ovpn's references by
	// base name (ADR-0105).
	Files []importFile `json:"files,omitempty"`
}

// importFile is one file added by hand to an import.
type importFile struct {
	Name          string `json:"name"`
	ContentBase64 string `json:"content_base64"`
}

// importBodyLimit: the .ovpn plus its added files, base64 (each at most
// MaxSize in total, so 2 × 512 KiB × 4/3 plus JSON fits).
const importBodyLimit = 2 << 20

// ErrKeyElevatedReadFailed: the permission prompt was dismissed or the
// elevated read failed (the "what to do" line offers the manual copy).
const ErrKeyElevatedReadFailed = "api.err.import_elevated_failed"

// importPath is the route; secureHandler does not hold the config lock for
// it (the handler locks only to store the file).
const importPath = "/api/v1/vpns/import"

// elevatedImportDeadline outlives the elevated read's own 3 min bound
// (cmd/kongtrol/protected_read.go) so its answer still reaches the page.
const elevatedImportDeadline = 4 * time.Minute

// importReadFile reads a "path" import (a variable so tests can answer
// "permission denied" on any OS).
var importReadFile = readRegularFile

// errNotRegular: the path is not (or stopped being) a plain file.
var errNotRegular = errors.New("not a regular file")

// readRegularFile reads path only if it is a regular file, not a link,
// and the file opened is the one checked (no swap in between), up to
// profileimport.MaxSize, through the opened descriptor.
func readRegularFile(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errNotRegular
	}
	f, err := os.OpenFile(path, os.O_RDONLY|openNoFollow, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, errNotRegular
	}
	if after.Size() > profileimport.MaxSize {
		return nil, profileimport.TooBig()
	}
	b, err := io.ReadAll(io.LimitReader(f, profileimport.MaxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > profileimport.MaxSize {
		return nil, profileimport.TooBig()
	}
	return b, nil
}

// cleanImportPath accepts an absolute path to a file the user picked and
// returns it cleaned. Refused: relative paths, any ".." element, NUL, and
// device or network namespaces (\\.\, \\?\, UNC shares, /dev, /proc,
// /sys), which are never a profile file and a UNC path would make Windows
// contact another machine.
func cleanImportPath(raw string) (string, bool) {
	p := strings.TrimSpace(raw)
	if p == "" || strings.ContainsRune(p, 0) || !filepath.IsAbs(p) {
		return "", false
	}
	for _, el := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if el == ".." {
			return "", false
		}
	}
	// NT object paths (\??\UNC\…, \??\GLOBALROOT\…) reach SMB or devices
	// too, and Windows treats them as absolute.
	for _, prefix := range []string{`\\`, "//", `/\`, `\/`, `\??\`, `/??/`} {
		if strings.HasPrefix(p, prefix) {
			return "", false
		}
	}
	if !plainLocalVolume(p) {
		return "", false
	}
	p = filepath.Clean(p)
	for _, dir := range []string{"/dev", "/proc", "/sys"} {
		if p == dir || strings.HasPrefix(p, dir+"/") {
			return "", false
		}
	}
	return p, true
}

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
	var refDir string
	var err error
	if strings.TrimSpace(req.Path) != "" {
		if req.Elevate {
			// The OS permission prompt may stay open for minutes: this
			// answer alone may outlive the server's 30 s WriteTimeout.
			_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(elevatedImportDeadline))
		}
		// Read without the config lock (secureHandler does not take it
		// for this route): a pending prompt must not block every other
		// change.
		if content, refDir, err = s.readImportPath(r.Context(), &req); err != nil {
			writeLocalizedError(w, r, importStatus(err), err)
			return
		}
	} else if content, err = base64.StdEncoding.DecodeString(strings.TrimSpace(req.ContentBase64)); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.import_content"})
		return
	}
	refs, err := importRefs(refDir, req.Files)
	if err != nil {
		writeLocalizedError(w, r, importStatus(err), err)
		return
	}
	dir, err := s.configsDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.configMu.Lock() // only to store the file (name taken / overwrite)
	res, err := profileimport.Import(dir, profileimport.Request{
		Name: req.Name, Type: req.Type, Content: content, Overwrite: req.Overwrite, Refs: refs,
	})
	s.configMu.Unlock()
	if err != nil {
		writeImportError(w, r, err)
		return
	}
	body := map[string]any{
		"status": "imported", "path": res.Path, "type": res.Type, "overwritten": res.Overwritten,
	}
	if len(res.Included) > 0 {
		body["included"] = res.Included
	}
	// A file from a provider account (Proton) says once that it clashes
	// with that provider's app (ADR-0103).
	if res.Provider != "" {
		body["provider"] = res.Provider
	}
	if key := profileimport.HintKey(res.Provider); key != "" {
		body["hint_key"] = key
		body["hint"] = i18n.T(requestLang(r), key)
	}
	writeJSON(w, http.StatusCreated, body)
}

// importRefs decodes the files added by hand and says where the .ovpn's
// references come from: dir (the picked file's folder, "" when there is
// none) and those files (ADR-0105).
func importRefs(dir string, files []importFile) (*profileimport.Refs, error) {
	if len(files) > profileimport.MaxExtraFiles {
		return nil, &config.ValidationError{Key: profileimport.ErrKeyTooManyFiles, Args: []any{profileimport.MaxExtraFiles}}
	}
	m := make(map[string][]byte, len(files))
	for _, f := range files {
		b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(f.ContentBase64))
		if err != nil {
			return nil, &config.ValidationError{Key: "api.err.import_content"}
		}
		m[f.Name] = b
	}
	return profileimport.NewRefs(dir, m)
}

// writeImportError is writeLocalizedError plus, for files the .ovpn needs
// and Kongtrol could not find, "missing": their names, so the panel can
// offer "Agregar <name>" for each (ADR-0105).
func writeImportError(w http.ResponseWriter, r *http.Request, err error) {
	var miss *profileimport.MissingFilesError
	if !errors.As(err, &miss) {
		writeLocalizedError(w, r, importStatus(err), err)
		return
	}
	var ve *config.ValidationError
	errors.As(err, &ve)
	lang := requestLang(r)
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"error":     ve.Message(lang),
		"hint":      ve.Hint(lang),
		"error_key": ve.Key,
		"missing":   miss.Names,
	})
}

// readImportPath reads req.Path and fills Type and Name from it when empty.
// The caller is the signed-in user of this machine (secureHandler) picking
// one of their own files, so any readable path is theirs to choose; still
// only an absolute, clean path (cleanImportPath) to a regular file (no
// link, directory or device) with a .conf / .ovpn name, read through the
// descriptor checked (readRegularFile). A root-only file needs Elevate.
// refDir is the file's folder, where the files an .ovpn names are looked
// for (ADR-0105); "" for a root-only file (the elevated read releases only
// a self-contained profile).
func (s *Server) readImportPath(ctx context.Context, req *importReq) (content []byte, refDir string, err error) {
	path, ok := cleanImportPath(req.Path)
	if !ok {
		return nil, "", &config.ValidationError{Key: profileimport.ErrKeyProtectedPath, Args: []any{strings.TrimSpace(req.Path)}}
	}
	elevated := false
	content, err = s.readImportFile(ctx, req, path, &elevated)
	if err != nil {
		return nil, "", err
	}
	if !elevated {
		refDir = filepath.Dir(path)
	}
	return content, refDir, nil
}

func (s *Server) readImportFile(ctx context.Context, req *importReq, path string, elevated *bool) ([]byte, error) {
	refuse := &config.ValidationError{Key: profileimport.ErrKeyProtectedPath, Args: []any{strings.TrimSpace(req.Path)}}
	typ := profileimport.TypeForFile(path)
	if typ == "" {
		return nil, refuse
	}
	// The type the caller left empty comes from the extension now and, once
	// the file is read, from its content (a .conf may be OpenVPN, ADR-0080).
	derived := req.Type == ""
	if derived {
		req.Type = typ
	}
	detect := func(b []byte) []byte {
		if derived {
			req.Type = profileimport.TypeForContent(path, b)
		}
		return b
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = profileimport.CleanName(path)
	}
	fi, err := os.Lstat(path)
	if err == nil {
		if !fi.Mode().IsRegular() {
			return nil, refuse
		}
		if fi.Size() > profileimport.MaxSize {
			return nil, profileimport.TooBig()
		}
		var b []byte
		if b, err = importReadFile(path); err == nil {
			return detect(b), nil
		}
		var ve *config.ValidationError
		if errors.As(err, &ve) {
			return nil, err
		}
	}
	if !config.IsPermission(err) {
		return nil, refuse
	}
	if !req.Elevate || s.readProtected == nil {
		return nil, &config.ValidationError{Key: config.ErrKeyFileRootOnly, Args: []any{path}}
	}
	*elevated = true
	b, err := s.readProtected(ctx, path)
	if err != nil {
		var ve *config.ValidationError
		if errors.As(err, &ve) {
			return nil, err
		}
		return nil, &config.ValidationError{Key: ErrKeyElevatedReadFailed, Args: []any{path}}
	}
	return detect(b), nil
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
	case "api.err.import_too_big", profileimport.ErrKeyTooManyFiles:
		return http.StatusRequestEntityTooLarge
	case config.ErrKeyFileRootOnly, ErrKeyElevatedReadFailed:
		return http.StatusForbidden
	}
	return http.StatusBadRequest
}
