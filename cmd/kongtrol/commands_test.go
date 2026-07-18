package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type mockAdapter struct {
	status vpn.Status
}

func (m *mockAdapter) Connect(context.Context, vpn.AdapterConfig) error { return nil }
func (m *mockAdapter) Disconnect(context.Context) error                 { return nil }
func (m *mockAdapter) Reconnect(context.Context) error                  { return nil }
func (m *mockAdapter) Status() vpn.Status                               { return m.status }
func (m *mockAdapter) TunnelInfo() (*vpn.TunnelInfo, error)             { return nil, nil }
func (m *mockAdapter) Name() string                                     { return "mock" }
func (m *mockAdapter) Version() string                                  { return "v0" }
func (m *mockAdapter) Capabilities() vpn.Capabilities                   { return vpn.Capabilities{} }

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	_ = r.Close()
	return string(out), runErr
}

func TestPolicyExplain_JSON(t *testing.T) {
	prevJSON := outputJSON
	prevEngine := engine
	outputJSON = true
	defer func() {
		outputJSON = prevJSON
		engine = prevEngine
	}()

	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{
			"office": {Type: "openvpn"},
		},
		Policies: []config.PolicyRule{
			{
				Name: "office-net",
				Match: config.MatchSpec{
					IPRanges: []string{"10.10.0.0/16"},
				},
				Via: "office",
			},
		},
	}
	e, err := policy.New(cfg)
	if err != nil {
		t.Fatalf("policy.New: %v", err)
	}
	engine = e

	out, runErr := captureStdout(t, func() error {
		return policyExplainCmd.RunE(policyExplainCmd, []string{"10.10.5.8"})
	})
	if runErr != nil {
		t.Fatalf("policy explain run: %v", runErr)
	}

	var got policy.ExplainResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("decode json: %v\noutput=%s", err, out)
	}
	if !got.Matched || got.Via != "office" || got.RuleName != "office-net" {
		t.Fatalf("unexpected explain result: %#v", got)
	}
}

func TestUpDryRun_JSON(t *testing.T) {
	prevJSON := outputJSON
	prevCfg := cfg
	prevAdapters := adapters
	outputJSON = true
	defer func() {
		outputJSON = prevJSON
		cfg = prevCfg
		adapters = prevAdapters
	}()

	cfg = &config.Config{
		VPNs: map[string]config.VPNConfig{
			"office": {Type: "openvpn"},
		},
	}
	adapters = map[string]vpn.VPNAdapter{
		"office": &mockAdapter{status: vpn.StatusDisconnected},
	}

	out, err := captureStdout(t, func() error {
		return runUpDryRun([]string{"office"})
	})
	if err != nil {
		t.Fatalf("runUpDryRun: %v", err)
	}

	var got dryRunReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("decode json: %v\noutput=%s", err, out)
	}
	if got.Failures != 0 {
		t.Fatalf("expected 0 failures, got %d", got.Failures)
	}
	if len(got.Profiles) == 0 {
		t.Fatalf("expected profile entries, got none")
	}
}

func TestDoctor_JSON(t *testing.T) {
	prevJSON := outputJSON
	prevCfgPath := cfgPath
	prevAdapters := adapters
	prevRouteMgr := routeMgr
	prevDNSMgr := dnsMgr
	outputJSON = true
	defer func() {
		outputJSON = prevJSON
		cfgPath = prevCfgPath
		adapters = prevAdapters
		routeMgr = prevRouteMgr
		dnsMgr = prevDNSMgr
	}()

	cfgPath = filepath.Join(t.TempDir(), "missing-kongtrol.yaml")
	adapters = map[string]vpn.VPNAdapter{}
	routeMgr = nil
	dnsMgr = nil

	out, err := captureStdout(t, func() error {
		return doctorCmd.RunE(doctorCmd, nil)
	})
	if err == nil {
		t.Fatalf("expected doctor to fail with missing config")
	}

	var got doctorReport
	if jerr := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); jerr != nil {
		t.Fatalf("decode json: %v\noutput=%s", jerr, out)
	}
	if got.Failures == 0 {
		t.Fatalf("expected failures in doctor report, got %#v", got)
	}
}
