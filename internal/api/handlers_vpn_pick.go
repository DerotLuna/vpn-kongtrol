package api

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/filepicker"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// importPickPath opens the OS's own "Open file" dialog (ADR-0105).
// secureHandler does not hold the config lock for it: the dialog may stay
// open for minutes.
const importPickPath = "/api/v1/vpns/import/pick"

// importPickFolderPath opens the same dialog in folder mode (ADR-0112).
const importPickFolderPath = "/api/v1/vpns/import/pick-folder"

// pickDeadline bounds how long the dialog may stay open (and this answer).
const pickDeadline = 10 * time.Minute

// Picker is the native dialog (a variable on the server so tests never
// open a window).
type Picker struct {
	Available  func() bool
	Pick       func(ctx context.Context, o filepicker.Options) (string, error)
	PickFolder func(ctx context.Context, o filepicker.Options) (string, error)
}

// DefaultPicker is the real dialog (internal/filepicker).
var DefaultPicker = Picker{Available: filepicker.Available, Pick: filepicker.Pick, PickFolder: filepicker.PickFolder}

// SetPicker replaces the native dialog (tests).
func (s *Server) SetPicker(p Picker) { s.picker = &p }

func (s *Server) filePicker() Picker {
	if s.picker != nil {
		return *s.picker
	}
	return DefaultPicker
}

// GET /api/v1/vpns/import/pick — {"available": bool}: whether "Elegir
// archivo" can open the OS's own dialog in this session; otherwise the
// panel uses the browser's file input.
func (s *Server) handleImportPickAvailable(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"available": s.filePicker().Available()})
}

// POST /api/v1/vpns/import/pick — opens the OS's own "Open file" dialog
// for a .ovpn / .conf and answers {"path", "name", "content_base64"} for
// the file picked (the panel shows it, then imports it with "path", so the
// files the .ovpn names are read from its folder), or {"cancelled": true}.
// The file is read like a "path" import (regular file, no link, size cap,
// the user's own rights); nothing of it is logged or audited. 503
// api.err.picker_unavailable (the panel falls back to the browser's
// input), 409 api.err.picker_busy (a dialog is already open).
func (s *Server) handleImportPick(w http.ResponseWriter, r *http.Request) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(pickDeadline + time.Minute))
	ctx, cancel := context.WithTimeout(r.Context(), pickDeadline)
	defer cancel()
	lang := requestLang(r)
	path, err := s.filePicker().Pick(ctx, filepicker.Options{
		Title:      i18n.T(lang, "picker.title"),
		FilterName: i18n.T(lang, "picker.filter"),
		Patterns:   []string{"*.ovpn", "*.conf"},
	})
	switch {
	case errors.Is(err, filepicker.ErrCanceled), err != nil && ctx.Err() != nil:
		writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
		return
	case errors.Is(err, filepicker.ErrBusy):
		writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "api.err.picker_busy"})
		return
	case err != nil:
		writeLocalizedError(w, r, http.StatusServiceUnavailable, &config.ValidationError{Key: "api.err.picker_unavailable"})
		return
	}
	req := importReq{Path: path}
	content, _, err := s.readImportPath(ctx, &req)
	if err != nil {
		writeLocalizedError(w, r, importStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":           path,
		"name":           filepath.Base(path),
		"type":           req.Type,
		"content_base64": base64.StdEncoding.EncodeToString(content),
	})
}

// POST /api/v1/vpns/import/pick-folder — opens the OS's own dialog in
// folder mode ("Elegir la carpeta donde están": the files a dropped .ovpn
// names are still missing) and answers {"path", "name"} for the folder
// picked, or {"cancelled": true}. The panel then imports again with
// "ref_dir": path. Only a clean local folder (no link, no junction, no
// network share) is answered; nothing in it is read here, and the path is
// never logged or audited (audit action config.profile.import_pick_folder:
// who / what / result). Same 503 / 409 as the file dialog (ADR-0112).
func (s *Server) handleImportPickFolder(w http.ResponseWriter, r *http.Request) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(pickDeadline + time.Minute))
	ctx, cancel := context.WithTimeout(r.Context(), pickDeadline)
	defer cancel()
	lang := requestLang(r)
	pick := s.filePicker().PickFolder
	if pick == nil {
		writeLocalizedError(w, r, http.StatusServiceUnavailable, &config.ValidationError{Key: "api.err.picker_unavailable"})
		return
	}
	path, err := pick(ctx, filepicker.Options{Title: i18n.T(lang, "picker.folder_title")})
	switch {
	case errors.Is(err, filepicker.ErrCanceled), err != nil && ctx.Err() != nil:
		writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
		return
	case errors.Is(err, filepicker.ErrBusy):
		writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "api.err.picker_busy"})
		return
	case err != nil:
		writeLocalizedError(w, r, http.StatusServiceUnavailable, &config.ValidationError{Key: "api.err.picker_unavailable"})
		return
	}
	dir, err := cleanRefDir(path)
	if err != nil {
		writeLocalizedError(w, r, importStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": dir, "name": filepath.Base(dir)})
}
