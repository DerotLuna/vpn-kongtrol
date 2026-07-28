package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"gopkg.in/yaml.v3"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) loadRuntimeConfig() (*config.Config, string, error) {
	cfgPath := s.configPath
	if cfgPath == "" {
		for _, candidate := range config.DefaultPaths() {
			if _, err := os.Stat(candidate); err == nil {
				cfgPath = candidate
				break
			}
		}
	}
	if cfgPath == "" {
		return nil, "", fmt.Errorf("config path not found")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, "", err
	}
	return cfg, cfgPath, nil
}

func (s *Server) saveRuntimeConfig(cfgPath string, cfg *config.Config) error {
	if err := config.Validate(cfg); err != nil {
		return err
	}
	newEngine, err := policy.New(cfg)
	if err != nil {
		return fmt.Errorf("policy engine validation failed: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := config.WriteFileAtomic(cfgPath, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	s.policyEngine.Store(newEngine)
	if s.onPolicyUpdate != nil {
		s.onPolicyUpdate(cfg, newEngine)
	}
	return nil
}
