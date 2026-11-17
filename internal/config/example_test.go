package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestExampleConfigLoads keeps configs/example.yaml usable: it must load
// and pass Validate exactly like a user's kongtrol.yaml, and every key in
// it must be a real schema field (strict decoding rejects typos and fields
// that were renamed or removed).
func TestExampleConfigLoads(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "example.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("configs/example.yaml does not load: %v", err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("configs/example.yaml does not validate: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dec := yaml.NewDecoder(bytes.NewReader([]byte(os.ExpandEnv(string(raw)))))
	dec.KnownFields(true)
	var strict Config
	if err := dec.Decode(&strict); err != nil {
		t.Fatalf("configs/example.yaml has a key the schema does not know: %v", err)
	}
}
