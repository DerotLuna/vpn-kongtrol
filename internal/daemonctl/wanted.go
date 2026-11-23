package daemonctl

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// Restore at start (ADR-0098): the daemon keeps the profiles the user
// wants up — connected by the user (panel, tray, `kongtrol up`) and not
// turned off by the user since — in ~/.kongtrol/run/wanted.json. It is
// written on every change (atomic replace + fsync), so a crash or a power
// cut keeps the last state; `kongtrol down` and "Salir y desconectar
// todo" empty it. A daemon that stops because the computer shuts down or
// signs out leaves it as it was, and the next start reconnects those
// profiles when "Al iniciar, volver a conectar" is on.
//
// Unlike resume.json (ADR-0050: one-shot, any setting, 24 h), this file
// is never consumed by reading it.

// WantedPath returns ~/.kongtrol/run/wanted.json.
func WantedPath() string { return filepath.Join(StateDir(), "run", "wanted.json") }

type wantedFile struct {
	Version   int       `json:"version"`
	Profiles  []string  `json:"profiles"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Wanted is the persisted set of profiles the user wants up.
type Wanted struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
}

// NewWanted keeps the set at path (WantedPath in the daemon).
func NewWanted(path string) *Wanted {
	return &Wanted{path: path, now: time.Now}
}

// List returns the saved profiles (sorted). A missing, unreadable or
// unknown file is an empty set: the worst case is not reconnecting.
func (w *Wanted) List() []string {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.read()
}

// Add records that the user turned name on.
func (w *Wanted) Add(name string) error {
	return w.update(func(set []string) []string {
		if slices.Contains(set, name) {
			return set
		}
		return append(set, name)
	})
}

// Remove records that the user turned name off.
func (w *Wanted) Remove(name string) error {
	return w.update(func(set []string) []string {
		return slices.DeleteFunc(set, func(n string) bool { return n == name })
	})
}

// Clear empties the set (`kongtrol down`, "Salir y desconectar todo",
// "Borrar todos mis datos").
func (w *Wanted) Clear() error {
	return w.update(func([]string) []string { return nil })
}

// Keep drops the profiles for which exists is false (deleted or renamed
// in the config) and returns what is left.
func (w *Wanted) Keep(exists func(string) bool) ([]string, error) {
	var kept []string
	err := w.update(func(set []string) []string {
		kept = slices.DeleteFunc(set, func(n string) bool { return !exists(n) })
		return kept
	})
	return kept, err
}

func (w *Wanted) update(change func([]string) []string) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	before := w.read()
	after := cleanProfiles(change(slices.Clone(before)))
	if slices.Equal(before, after) {
		return nil
	}
	if len(after) == 0 {
		if err := os.Remove(w.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	data, err := json.Marshal(wantedFile{Version: 1, Profiles: after, UpdatedAt: w.now().UTC()})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(w.path), 0o700); err != nil {
		return err
	}
	return config.WriteFileAtomic(w.path, data, 0o600)
}

func (w *Wanted) read() []string {
	f, err := os.Open(w.path)
	if err != nil {
		return nil
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64<<10))
	if err != nil {
		return nil
	}
	var wf wantedFile
	if json.Unmarshal(data, &wf) != nil || wf.Version != 1 {
		return nil
	}
	return cleanProfiles(wf.Profiles)
}
