package configexport

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// fullConfig uses the fields the old hand-written export lost or broke:
// a policy with both IPs and domains (duplicate "match:"), a glob domain
// (unquoted "*.corp.com" is a YAML alias), lists ("[a b]"), provider
// fields, failover, on_tunnel_down, security and scheduler settings.
const fullConfig = `vpns:
  office:
    type: openvpn
    config: /etc/kongtrol/office.ovpn
    priority: 2
    failover: [backup]
    kill_switch: false
    auth:
      method: credentials
      username: ana
      password_keychain: office
  backup:
    type: wireguard
    config: /etc/kongtrol/backup.conf
    auth:
      method: none
  proton:
    type: protonvpn
    country: CH
    auto_connect: false
    wait_timeout: 2m
    interface: "proton*"
groups:
  work:
    profiles: [office, backup]
policies:
  - name: corp
    match:
      ip_ranges: [10.0.0.0/8, 172.16.0.0/12]
      domains: ["*.corp.com", intranet.example.com]
    via: office
    on_tunnel_down: block
  - name: streaming
    match:
      domains: ["*.example.tv"]
    via: proton
security:
  kill_switch:
    enabled: true
    allow_lan: true
  dns_guard:
    enabled: true
    fallback_dns: 1.1.1.1
  public_ip_check:
    enabled: true
    endpoints: [https://api.ipify.org]
monitor:
  dashboard:
    port: 9800
  scheduler:
    enabled: true
    rules:
      - name: weekdays
        profiles: [office]
        weekdays: [mon, tue]
        start: "08:00"
        end: "18:00"
`

func load(t *testing.T, body string) *config.Config {
	t.Helper()
	p := filepath.Join(t.TempDir(), "kongtrol.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(p)
	if err != nil {
		t.Fatalf("load: %v\n%s", err, body)
	}
	return cfg
}

func export(t *testing.T, cfg *config.Config, lang i18n.Lang) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Write(&buf, cfg, lang, Style{}); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestRoundTripThroughLoader(t *testing.T) {
	orig := load(t, fullConfig)
	out := export(t, orig, i18n.ES)
	back := load(t, out)
	if !reflect.DeepEqual(orig, back) {
		t.Fatalf("export does not load back to the same config:\n%s", out)
	}
	// Exporting the re-loaded config is byte-identical (stable order).
	if again := export(t, back, i18n.ES); again != out {
		t.Fatalf("unstable output:\n%s\n---\n%s", out, again)
	}
}

func TestOutputShape(t *testing.T) {
	out := export(t, load(t, fullConfig), i18n.EN)
	for _, want := range []string{
		`- '*.corp.com'`, // quoted glob
		"- 10.0.0.0/8",   // a real YAML list, not "[10.0.0.0/8 …]"
		"on_tunnel_down: block",
		"auto_connect: false",
		"country: CH",
		"failover:",
		"fallback_dns: 1.1.1.1",
		"password_keychain: office",
		"# " + i18n.T(i18n.EN, "export.header.secrets"),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "match:"); n != 2 {
		t.Errorf("want one match: per policy (2), got %d", n)
	}
	if strings.Contains(out, "[10.0.0.0/8") {
		t.Errorf("Go-formatted list leaked:\n%s", out)
	}
}

func TestStyleKeepsText(t *testing.T) {
	cfg := load(t, fullConfig)
	tag := func(p string) func(string) string { return func(s string) string { return "<" + p + ">" + s + "</>" } }
	var buf bytes.Buffer
	if err := Write(&buf, cfg, i18n.ES, Style{Section: tag("s"), Key: tag("k"), Value: tag("v"), Comment: tag("c")}); err != nil {
		t.Fatal(err)
	}
	styled := buf.String()
	for _, want := range []string{"<s>vpns</><k>:</>", "<k>type</><k>:</> <v>openvpn</>", "<c># "} {
		if !strings.Contains(styled, want) {
			t.Errorf("missing %q in:\n%s", want, styled)
		}
	}
	strip := strings.NewReplacer("<s>", "", "<k>", "", "<v>", "", "<c>", "", "</>", "")
	if strip.Replace(styled) != export(t, cfg, i18n.ES) {
		t.Fatal("styling must only wrap text, never change it")
	}
}

func TestNilConfig(t *testing.T) {
	if err := Write(&bytes.Buffer{}, nil, i18n.EN, Style{}); err == nil {
		t.Fatal("nil config must fail")
	}
}
