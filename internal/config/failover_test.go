package config

import (
	"errors"
	"sort"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestValidateFailover(t *testing.T) {
	cases := []struct {
		name    string
		vpns    map[string]VPNConfig
		wantKey string // "" = valid
	}{
		{"none declared", map[string]VPNConfig{"a": {}, "b": {}}, ""},
		{"chain", map[string]VPNConfig{"a": {Failover: []string{"b"}}, "b": {Failover: []string{"c"}}, "c": {}}, ""},
		{"two backups", map[string]VPNConfig{"a": {Failover: []string{"b", "c"}}, "b": {}, "c": {}}, ""},
		{"unknown", map[string]VPNConfig{"a": {Failover: []string{"nope"}}}, "config.err.failover_unknown"},
		{"self", map[string]VPNConfig{"a": {Failover: []string{"a"}}}, "config.err.failover_self"},
		{"duplicate", map[string]VPNConfig{"a": {Failover: []string{"b", "b"}}, "b": {}}, "config.err.failover_duplicate"},
		{"cycle of two", map[string]VPNConfig{"a": {Failover: []string{"b"}}, "b": {Failover: []string{"a"}}}, "config.err.failover_cycle"},
		{"cycle of three", map[string]VPNConfig{"a": {Failover: []string{"b"}}, "b": {Failover: []string{"c"}}, "c": {Failover: []string{"a"}}}, "config.err.failover_cycle"},
	}
	for _, c := range cases {
		names := make([]string, 0, len(c.vpns))
		for n := range c.vpns {
			names = append(names, n)
		}
		sort.Strings(names)
		err := validateFailover(c.vpns, names)
		var ve *ValidationError
		switch {
		case c.wantKey == "" && err != nil:
			t.Errorf("%s: unexpected error %v", c.name, err)
		case c.wantKey != "" && (!errors.As(err, &ve) || ve.Key != c.wantKey):
			t.Errorf("%s: err = %v, want key %s", c.name, err, c.wantKey)
		}
		if ve != nil {
			for _, l := range []i18n.Lang{i18n.ES, i18n.EN} {
				if msg, hint := ve.Message(l), ve.Hint(l); msg == ve.Key || hint == ve.Key+".fix" {
					t.Errorf("%s: missing %s translation (%q / %q)", c.name, l, msg, hint)
				}
			}
		}
	}
}

func TestValidateFailover_CycleMessage(t *testing.T) {
	err := validateFailover(map[string]VPNConfig{
		"a": {Failover: []string{"b"}}, "b": {Failover: []string{"a"}},
	}, []string{"a", "b"})
	if err == nil || err.Error() != "failover profiles form a loop: a → b → a" {
		t.Fatalf("err = %v", err)
	}
}
