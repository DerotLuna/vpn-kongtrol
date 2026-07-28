package main

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAutoScalar(t *testing.T) {
	cases := []struct {
		in      string
		wantTag string
		wantVal string
	}{
		{"true", "!!bool", "true"},
		{"True", "!!bool", "true"},
		{"false", "!!bool", "false"},
		{"42", "!!int", "42"},
		{"hello", "!!str", "hello"},
		{"", "!!str", ""},
	}
	for _, tc := range cases {
		n := autoScalar(tc.in)
		if n.Tag != tc.wantTag || n.Value != tc.wantVal {
			t.Errorf("autoScalar(%q) = {%s %s}, want {%s %s}", tc.in, n.Tag, n.Value, tc.wantTag, tc.wantVal)
		}
	}
}

func TestMapNode(t *testing.T) {
	n := mapNode([][2]string{{"host", "vpn.example.com"}, {"port", "443"}})
	if n.Kind != yaml.MappingNode {
		t.Fatalf("expected MappingNode, got %v", n.Kind)
	}
	if len(n.Content) != 4 {
		t.Fatalf("expected 4 content nodes (2 pairs), got %d", len(n.Content))
	}
	if v := mappingKey(n, "host"); v == nil || v.Value != "vpn.example.com" {
		t.Errorf("mappingKey(host) = %v, want vpn.example.com", v)
	}
	if v := mappingKey(n, "port"); v == nil || v.Value != "443" || v.Tag != "!!int" {
		t.Errorf("mappingKey(port) = %v, want int 443", v)
	}
	if v := mappingKey(n, "missing"); v != nil {
		t.Errorf("mappingKey(missing) = %v, want nil", v)
	}
}

func TestMappingKeyOnNil(t *testing.T) {
	if v := mappingKey(nil, "anything"); v != nil {
		t.Errorf("mappingKey(nil, ...) = %v, want nil", v)
	}
}

func TestSetMapping(t *testing.T) {
	n := mapNode([][2]string{{"a", "1"}})
	setMapping(n, "a", scalarNode("2"))
	if v := mappingKey(n, "a"); v == nil || v.Value != "2" {
		t.Errorf("setMapping did not overwrite existing key: got %v", v)
	}
	setMapping(n, "b", scalarNode("3"))
	if v := mappingKey(n, "b"); v == nil || v.Value != "3" {
		t.Errorf("setMapping did not append new key: got %v", v)
	}
}

func TestRemoveMapping(t *testing.T) {
	n := mapNode([][2]string{{"a", "1"}, {"b", "2"}})
	removeMapping(n, "a")
	if v := mappingKey(n, "a"); v != nil {
		t.Errorf("removeMapping did not remove key: got %v", v)
	}
	if v := mappingKey(n, "b"); v == nil || v.Value != "2" {
		t.Errorf("removeMapping removed the wrong key: got %v", v)
	}
	// Removing a missing key or on a nil parent must not panic.
	removeMapping(n, "missing")
	removeMapping(nil, "a")
}

func TestPolicyNodeAndRemovePolicyByName(t *testing.T) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	seq.Content = append(seq.Content,
		policyNode("work", "office-vpn", []string{"*.corp"}, nil),
		policyNode("streaming", "exit-node", nil, []string{"10.0.0.0/8"}),
	)
	if len(seq.Content) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(seq.Content))
	}

	first := seq.Content[0]
	if v := mappingKey(first, "name"); v == nil || v.Value != "work" {
		t.Errorf("policy name = %v, want work", v)
	}
	if v := mappingKey(first, "via"); v == nil || v.Value != "office-vpn" {
		t.Errorf("policy via = %v, want office-vpn", v)
	}
	match := mappingKey(first, "match")
	if match == nil || mappingKey(match, "domains") == nil {
		t.Error("expected match.domains to be set")
	}

	removePolicyByName(seq, "work")
	if len(seq.Content) != 1 {
		t.Fatalf("expected 1 policy after removal, got %d", len(seq.Content))
	}
	if v := mappingKey(seq.Content[0], "name"); v == nil || v.Value != "streaming" {
		t.Errorf("remaining policy = %v, want streaming", v)
	}

	// Removing a name that doesn't exist, or on a nil sequence, must not panic.
	removePolicyByName(seq, "missing")
	removePolicyByName(nil, "work")
}

func TestFreshDoc(t *testing.T) {
	doc := freshDoc()
	if doc.Kind != yaml.MappingNode {
		t.Fatalf("expected MappingNode, got %v", doc.Kind)
	}
	if len(doc.Content) != 0 {
		t.Errorf("expected empty document, got %d content nodes", len(doc.Content))
	}
}

func TestLoadExistingConfigMissingFile(t *testing.T) {
	cfg, doc := loadExistingConfig("/nonexistent/path/kongtrol.yaml")
	if cfg != nil || doc != nil {
		t.Errorf("expected nil, nil for missing file, got %v, %v", cfg, doc)
	}
}
