// Package privhelper is Kongtrol's privileged helper service: installed once
// (`kongtrol service install`), it runs as a Windows service (LocalSystem),
// launchd daemon or systemd unit (root) and serves privops operations to
// the installing user over a local named pipe / Unix socket.
package privhelper

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// ConfigFormat is the helper.json schema version.
const ConfigFormat = 1

// Config is helper.json, written at install time to an admin/root-only
// location. It records who may talk to the helper.
type Config struct {
	Format      int       `json:"format"`
	UserID      string    `json:"user_id"` // Windows SID or Unix uid
	UserName    string    `json:"user_name"`
	GID         int       `json:"gid"` // Unix primary group (socket group); -1 on Windows
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

// ErrNotInstalled means helper.json is missing.
var ErrNotInstalled = errors.New("privhelper: not installed")

func (c Config) validate() error {
	if c.Format != ConfigFormat {
		return fmt.Errorf("privhelper: helper.json format %d unsupported", c.Format)
	}
	return validateUserID(c.UserID)
}

// LoadConfig reads helper.json after checking it is admin/root-owned and
// not writable by other users.
func LoadConfig() (Config, error) {
	path := ConfigPath()
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return Config{}, ErrNotInstalled
	}
	if err := checkConfigFileTrusted(path); err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := privops.DecodeStrict(b, &c); err != nil {
		return Config{}, fmt.Errorf("privhelper: helper.json: %w", err)
	}
	return c, c.validate()
}

func encodeConfig(c Config) ([]byte, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(c, "", "  ")
}
