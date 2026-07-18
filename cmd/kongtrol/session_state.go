package main

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

type cliSessionState struct {
	LastCommandAt time.Time `json:"last_command_at,omitempty"`
	LastCommand   string    `json:"last_command,omitempty"`

	// Legacy keys from early draft; kept for backwards compatibility.
	LastLoginAt   time.Time `json:"last_login_at,omitempty"`
	LastLoginFrom string    `json:"last_login_from,omitempty"`
}

func sessionStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "kongtrol-session.json"
	}
	return filepath.Join(home, ".kongtrol", "session.json")
}

func loadSessionState(path string) (cliSessionState, error) {
	if strings.TrimSpace(path) == "" {
		return cliSessionState{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cliSessionState{}, nil
		}
		return cliSessionState{}, err
	}
	var st cliSessionState
	if err := json.Unmarshal(b, &st); err != nil {
		return cliSessionState{}, err
	}
	return st, nil
}

func saveSessionState(path string, st cliSessionState) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func resolveSystemUserName() string {
	if u, err := user.Current(); err == nil {
		if name := strings.TrimSpace(u.Name); name != "" {
			return name
		}
		if name := strings.TrimSpace(u.Username); name != "" {
			return name
		}
	}
	if v := strings.TrimSpace(os.Getenv("USERNAME")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("USER")); v != "" {
		return v
	}
	return "user"
}
