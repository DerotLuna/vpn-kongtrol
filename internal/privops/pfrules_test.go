package privops

import (
	"errors"
	"net/netip"
	"strings"
	"testing"
)

const applePFConf = `scrub-anchor "com.apple/*"
nat-anchor "com.apple/*"
rdr-anchor "com.apple/*"
dummynet-anchor "com.apple/*"
anchor "com.apple/*"
load anchor "com.apple" from "/etc/pf.anchors/com.apple"`

func TestPFMainRulesetKeepsSystemAndAddsAnchorOnce(t *testing.T) {
	once := string(pfMainRuleset([]byte(applePFConf)))
	if !strings.HasPrefix(once, applePFConf+"\n") {
		t.Errorf("original pf.conf not preserved:\n%s", once)
	}
	twice := string(pfMainRuleset([]byte(once)))
	if twice != once {
		t.Errorf("not idempotent:\n%s\n---\n%s", once, twice)
	}
	if n := strings.Count(twice, `anchor "com.kongtrol"`); n != 1 {
		t.Errorf("anchor appears %d times", n)
	}
	if !strings.HasSuffix(twice, "anchor \"com.kongtrol\"\n") {
		t.Errorf("anchor must be the last filter rule:\n%s", twice)
	}
	if got := string(pfMainRuleset(nil)); got != pfAnchorComment+"\n"+pfAnchorLine+"\n" {
		t.Errorf("empty pf.conf: %q", got)
	}
}

func TestPFKillSwitchRules(t *testing.T) {
	req := KillSwitchRequest{
		Interfaces: []string{"utun4"},
		Endpoints:  []netip.Addr{netip.MustParseAddr("203.0.113.7"), netip.MustParseAddr("2001:db8::1")},
		AllowLAN:   true,
	}
	got, err := pfKillSwitchRules(req)
	if err != nil {
		t.Fatal(err)
	}
	want := `# Kongtrol kill switch
pass out quick on lo0 all
pass out quick on utun4 all
pass out quick inet proto { udp tcp } to 203.0.113.7
pass out quick inet6 proto { udp tcp } to 2001:db8::1
pass out quick inet proto udp from any port 68 to any port 67
pass out quick inet6 proto udp from any port 546 to any port 547
pass out quick inet6 proto icmp6 all icmp6-type { routersol neighbrsol neighbradv }
pass out quick inet to { 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16 169.254.0.0/16 224.0.0.0/24 255.255.255.255 }
pass out quick inet6 to { fe80::/10 fc00::/7 ff02::/16 }
block drop out quick all
`
	if got != want {
		t.Errorf("rules:\n%s\nwant:\n%s", got, want)
	}

	noLAN, _ := pfKillSwitchRules(KillSwitchRequest{})
	if strings.Contains(noLAN, "192.168.0.0/16") || !strings.HasSuffix(noLAN, "block drop out quick all\n") {
		t.Errorf("without LAN:\n%s", noLAN)
	}
	for _, line := range strings.Split(strings.TrimSpace(noLAN), "\n") {
		if !strings.HasPrefix(line, "#") && !strings.Contains(line, " quick ") {
			t.Errorf("rule without quick: %q", line)
		}
	}
}

func TestPFKillSwitchRulesRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{"en0#", "egress", "utun 1", "any", "en0)", "-en0", "UTUN1x", ""} {
		if _, err := pfKillSwitchRules(KillSwitchRequest{Interfaces: []string{name}}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%q accepted: %v", name, err)
		}
	}
	for _, ep := range []netip.Addr{{}, netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("fe80::1%en0"), netip.MustParseAddr("0.0.0.0")} {
		if _, err := pfKillSwitchRules(KillSwitchRequest{Endpoints: []netip.Addr{ep}}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("endpoint %v accepted: %v", ep, err)
		}
	}
}

func TestParsePFToken(t *testing.T) {
	out := "No ALTQ support in kernel\nALTQ related functions disabled\npf enabled\nToken : 13886405473364946027\n"
	if tok, err := parsePFToken(out); err != nil || tok != "13886405473364946027" {
		t.Errorf("token %q %v", tok, err)
	}
	if tok, err := parsePFToken("pf already enabled\nToken : 42"); err != nil || tok != "42" {
		t.Errorf("token %q %v", tok, err)
	}
	for _, bad := range []string{"", "pf enabled\n", "Token : -1", "Token : 1; rm"} {
		if _, err := parsePFToken(bad); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}
