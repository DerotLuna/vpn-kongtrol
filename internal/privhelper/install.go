package privhelper

import (
	"errors"
	"io"
	"os"
)

// InstallOptions describes one installation.
type InstallOptions struct {
	SourceExe string // executable to copy (the running kongtrol)
	UserID    string // installing user's SID (Windows) or uid (Unix)
	UserName  string
	GID       int // Unix: installing user's primary group (socket group)
	Version   string
}

// State is what `service status` reports.
type State struct {
	Installed  bool
	Running    bool
	BinaryPath string
}

// copyExecutable copies src to dst via a temp file in dst's directory, so
// a half-written binary is never registered.
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	_ = os.Remove(tmp)
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := chownRoot(tmp, 0o755); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// ErrElevationCancelled is returned when the user declines the UAC prompt.
var ErrElevationCancelled = errors.New("privhelper: administrator permission was not granted")
