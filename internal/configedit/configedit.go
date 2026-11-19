// Package configedit holds the kongtrol.yaml edits shared by the daemon's
// dashboard API (internal/api) and the CLI when no daemon is running
// (`kongtrol policy add|remove`, `kongtrol profile add`): one
// load → mutate → config.Validate → save path, so both check and store a
// change the same way.
package configedit

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

// Save validates cfg (struct tags + semantics, then the policy engine) and
// writes it atomically to path (0600). It returns the engine built from
// cfg, which a running daemon hot-swaps in.
func Save(path string, cfg *config.Config) (*policy.Engine, error) {
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	engine, err := policy.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("policy engine validation failed: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	if err := config.WriteFileAtomic(path, data, 0o600); err != nil {
		return nil, fmt.Errorf("write config: %w", err)
	}
	return engine, nil
}
