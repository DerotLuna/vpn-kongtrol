package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const apiTokenBytes = 32

// ValidateDashboardBind keeps the privileged local control plane on loopback.
func ValidateDashboardBind(bind string) error {
	bind = strings.TrimSpace(bind)
	if strings.EqualFold(strings.TrimSuffix(bind, "."), "localhost") {
		return nil
	}
	ip := net.ParseIP(bind)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("dashboard bind %q is not a loopback address", bind)
	}
	return nil
}

// LoadOrCreateAPIToken returns the per-user capability used by local API clients.
func LoadOrCreateAPIToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("api token: home directory: %w", err)
	}
	dir := filepath.Join(home, ".kongtrol")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("api token: create directory: %w", err)
	}
	path := filepath.Join(dir, "api-token")

	if token, err := readAPIToken(path); err == nil {
		return token, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	raw := make([]byte, apiTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("api token: generate: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		for range 10 {
			if token, readErr := readAPIToken(path); readErr == nil {
				return token, nil
			}
			time.Sleep(10 * time.Millisecond)
		}
		return "", fmt.Errorf("api token: concurrent creation did not complete")
	}
	if err != nil {
		return "", fmt.Errorf("api token: create: %w", err)
	}
	if _, err := f.WriteString(token); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("api token: write: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("api token: sync: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("api token: close: %w", err)
	}
	return token, nil
}

func readAPIToken(path string) (string, error) {
	if err := os.Chmod(path, 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("api token: secure permissions: %w", err)
	}
	return parseAPIToken(path)
}

func parseAPIToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != apiTokenBytes {
		return "", fmt.Errorf("api token: invalid token file %s", path)
	}
	return token, nil
}

// ReadAPIToken returns the existing per-user API token without creating one,
// for read-only callers (doctor) that only talk to an already-running daemon.
func ReadAPIToken() (string, error) {
	path, err := apiTokenPath()
	if err != nil {
		return "", err
	}
	return readAPIToken(path)
}

// RotateAPIToken writes a fresh per-user API token to ~/.kongtrol/api-token
// (0600, atomically: temp file + rename) and returns it. The daemon calls it
// on every start, before it listens, so a token sent to the wrong listener
// earlier stops working once Kongtrol starts again (ADR-0062). Clients
// re-read the file per request (CurrentAPIToken).
func RotateAPIToken() (string, error) {
	path, err := apiTokenPath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("api token: create directory: %w", err)
	}
	raw := make([]byte, apiTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("api token: generate: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	// Temp file + rename (MoveFileEx with retries on Windows, where a
	// client may be reading the old file at this very moment).
	if err := WriteFileAtomic(path, []byte(token), 0o600); err != nil {
		return "", fmt.Errorf("api token: %w", err)
	}
	return token, nil
}

// CurrentAPIToken reads ~/.kongtrol/api-token as it is now, for clients of a
// daemon that rotates it on every start. Unlike ReadAPIToken it never
// touches the file's permissions (it runs on every request).
func CurrentAPIToken() (string, error) {
	path, err := apiTokenPath()
	if err != nil {
		return "", err
	}
	return parseAPIToken(path)
}

func apiTokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("api token: home directory: %w", err)
	}
	return filepath.Join(home, ".kongtrol", "api-token"), nil
}
