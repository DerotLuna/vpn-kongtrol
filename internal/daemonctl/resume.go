package daemonctl

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Resume after an update (ADR-0050): an installer stops Kongtrol before it
// replaces the files (`kongtrol down --all --remember`), which records the
// profiles that were connected; once the new files are in place
// `kongtrol resume` reconnects exactly those. The file is one-shot: reading
// it deletes it, so a later start never reconnects anything by surprise.

// ResumeMaxAge bounds how old a resume file may be and still count: an
// update that stopped Kongtrol and then failed must not reconnect tunnels
// days later.
const ResumeMaxAge = 24 * time.Hour

// resumeMaxProfiles caps the list read back (the file is user-writable).
const resumeMaxProfiles = 256

type resumeFile struct {
	Version  int       `json:"version"`
	Profiles []string  `json:"profiles"`
	SavedAt  time.Time `json:"saved_at"`
}

// ResumePath returns ~/.kongtrol/run/resume.json.
func ResumePath() string { return filepath.Join(StateDir(), "run", "resume.json") }

// WriteResume records profiles (deduplicated, sorted) at path. An empty list
// removes any previous file instead, so nothing stale is resumed.
func WriteResume(path string, profiles []string, now time.Time) error {
	clean := cleanProfiles(profiles)
	if len(clean) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	data, err := json.Marshal(resumeFile{Version: 1, Profiles: clean, SavedAt: now.UTC()})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// TakeResume reads and deletes the resume file at path. It returns no
// profiles (and no error) when the file is missing, unreadable as JSON, of
// an unknown version or older than ResumeMaxAge.
func TakeResume(path string, now time.Time) ([]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(f, 64<<10))
	_ = f.Close()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if readErr != nil {
		return nil, readErr
	}
	var rf resumeFile
	if json.Unmarshal(data, &rf) != nil || rf.Version != 1 {
		return nil, nil
	}
	age := now.Sub(rf.SavedAt)
	if age < -time.Minute || age > ResumeMaxAge {
		return nil, nil
	}
	return cleanProfiles(rf.Profiles), nil
}

// cleanProfiles drops empty names, duplicates and anything past the cap.
func cleanProfiles(in []string) []string {
	out := make([]string, 0, len(in))
	for _, name := range in {
		if name == "" || slices.Contains(out, name) {
			continue
		}
		out = append(out, name)
		if len(out) == resumeMaxProfiles {
			break
		}
	}
	slices.Sort(out)
	return out
}
