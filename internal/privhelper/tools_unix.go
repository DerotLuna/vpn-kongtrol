//go:build linux || darwin

package privhelper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func toolsManifestPath() string { return filepath.Join(privops.ToolsDir, toolsManifest) }

func systemToolTrusted(paths ...string) bool {
	_, err := privops.TrustedBinary(paths...)
	return err == nil
}

func toolExists(p string) bool {
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return false
	}
	fi, err := os.Stat(real)
	return err == nil && fi.Mode().IsRegular()
}

// installTools copies the tools the helper cannot use in place into
// privops.ToolsDir (root:root 0755) and writes its manifest. Candidates
// come only from the fixed toolSpecs, never from the caller. A tool that
// cannot be copied safely is recorded as unusable; it does not fail the
// install. Files from earlier installs that are no longer needed are
// removed.
func installTools() error {
	dir := privops.ToolsDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := chownRoot(dir, 0o755); err != nil {
		return err
	}
	if !privops.TrustedDir(dir) {
		return fmt.Errorf("privhelper: %s or a parent is not root-owned; fix its ownership and retry", dir)
	}
	m := ToolManifest{Format: toolsFormat, Tools: map[string]ToolRecord{}, Unusable: map[string]ToolSkip{}}
	usable := map[string]bool{}
	for _, p := range planTools(toolSpecs, systemToolTrusted, toolExists) {
		if p.Requires != "" && !usable[p.Requires] {
			continue
		}
		if p.System {
			usable[p.Name] = true
			continue
		}
		rec, err := copyTool(p.Source, filepath.Join(dir, p.Name))
		if err != nil {
			m.Unusable[p.Name] = ToolSkip{Source: p.Source, Reason: err.Error()}
			continue
		}
		m.Tools[p.Name] = rec
		usable[p.Name] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if _, keep := m.Tools[e.Name()]; !keep && e.Name() != toolsManifest {
			_ = os.RemoveAll(filepath.Join(dir, e.Name()))
		}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return writeRootFile(filepath.Join(dir, toolsManifest), data, 0o644)
}

// copyTool reads src (symlinks resolved, opened once without following a
// final symlink), checks the bytes it read and installs them at dst. The
// recorded hash is of the source bytes, so `service status` can tell when
// the source is upgraded.
func copyTool(src, dst string) (ToolRecord, error) {
	real, err := filepath.EvalSymlinks(src)
	if err != nil {
		return ToolRecord{}, err
	}
	fd, err := unix.Open(real, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return ToolRecord{}, err
	}
	f := os.NewFile(uintptr(fd), real)
	defer f.Close()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return ToolRecord{}, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		return ToolRecord{}, fmt.Errorf("%s is not a regular file", real)
	}
	if st.Size > maxToolSize {
		return ToolRecord{}, fmt.Errorf("%s is larger than %d bytes", real, maxToolSize)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxToolSize+1))
	if err != nil {
		return ToolRecord{}, err
	}
	if len(data) > maxToolSize {
		return ToolRecord{}, fmt.Errorf("%s is larger than %d bytes", real, maxToolSize)
	}
	out, err := prepareToolContent(data)
	if err != nil {
		return ToolRecord{}, fmt.Errorf("%s: %w", real, err)
	}
	if err := writeRootFile(dst, out, 0o755); err != nil {
		return ToolRecord{}, err
	}
	sum := sha256.Sum256(data)
	return ToolRecord{Source: src, Resolved: real, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data))}, nil
}

// writeRootFile writes data to path atomically (temp file + rename) as
// root-owned with mode.
func writeRootFile(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".new"
	_ = os.Remove(tmp)
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	if _, err := out.Write(data); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := chownRoot(tmp, mode); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
