package main

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// ── YAML document helpers ─────────────────────────────────────────────────────
//
// These are pure functions over yaml.Node trees — they build or mutate the
// document the wizard writes to kongtrol.yaml. Kept free of huh/i18n so they
// stay unit-testable without a TTY.

func loadExistingConfig(path string) (*config.Config, *yaml.Node) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, nil
	}
	doc := &root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, doc
	}
	return &cfg, doc
}

func freshDoc() *yaml.Node {
	return mapNode([][2]string{})
}

func scalarNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"}
}

func boolNode(val bool) *yaml.Node {
	v := "false"
	if val {
		v = "true"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: "!!bool"}
}

func intNode(val string) *yaml.Node {
	if _, err := strconv.Atoi(val); err != nil {
		return scalarNode(val)
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!int"}
}

func mapNode(pairs [][2]string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, p := range pairs {
		n.Content = append(n.Content, scalarNode(p[0]), autoScalar(p[1]))
	}
	return n
}

func autoScalar(val string) *yaml.Node {
	switch strings.ToLower(val) {
	case "true":
		return boolNode(true)
	case "false":
		return boolNode(false)
	}
	if _, err := strconv.Atoi(val); err == nil {
		return intNode(val)
	}
	return scalarNode(val)
}

func mappingKey(n *yaml.Node, key string) *yaml.Node {
	if n == nil {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func removeMapping(parent *yaml.Node, key string) {
	if parent == nil {
		return
	}
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
			return
		}
	}
}

func setMapping(parent *yaml.Node, key string, val *yaml.Node) {
	if parent == nil {
		return
	}
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			parent.Content[i+1] = val
			return
		}
	}
	parent.Content = append(parent.Content, scalarNode(key), val)
}
