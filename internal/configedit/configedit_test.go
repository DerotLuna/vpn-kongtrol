package configedit

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

const testYAML = "vpns:\n  office:\n    type: wireguard\n    config: /etc/wg.conf\n    auth:\n      method: certificate\n"

func load(t *testing.T) (*config.Config, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	if err := os.WriteFile(path, []byte(testYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return c, path
}

func key(err error) string {
	var ve *config.ValidationError
	if errors.As(err, &ve) {
		return ve.Key
	}
	return ""
}

func TestAddRemovePolicy_SaveRoundTrip(t *testing.T) {
	c, path := load(t)
	rule := config.PolicyRule{Name: " corp ", Via: "office", OnTunnelDown: "Allow",
		Match: config.MatchSpec{Domains: []string{" a.com ", ""}, IPRanges: []string{"10.0.0.0/8"}}}
	got, err := AddPolicy(c, rule)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "corp" || got.OnTunnelDown != "" || len(got.Match.Domains) != 1 || rule.Match.Domains[0] != " a.com " {
		t.Fatalf("normalized = %+v (input changed: %q)", got, rule.Match.Domains)
	}
	if _, err := AddPolicy(c, config.PolicyRule{Name: "CORP", Via: "office", Match: config.MatchSpec{Domains: []string{"b.com"}}}); key(err) != ErrKeyPolicyExists {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	c2, err := config.Load(path)
	if err != nil || len(c2.Policies) != 1 {
		t.Fatalf("reloaded: %v %+v", err, c2)
	}
	if err := RemovePolicy(c2, "Corp"); err != nil || len(c2.Policies) != 0 {
		t.Fatalf("remove: %v", err)
	}
	if err := RemovePolicy(c2, "corp"); key(err) != ErrKeyPolicyNotFound {
		t.Fatalf("remove missing: %v", err)
	}
}

func TestValidatePolicy_Keys(t *testing.T) {
	c, _ := load(t)
	m := config.MatchSpec{Domains: []string{"a.com"}}
	cases := []struct {
		rule config.PolicyRule
		want string
	}{
		{config.PolicyRule{Via: "office", Match: m}, "api.err.name_required"},
		{config.PolicyRule{Name: "x", Match: m}, "api.err.policy_via_required"},
		{config.PolicyRule{Name: "x", Via: "nope", Match: m}, "api.err.policy_via_unknown"},
		{config.PolicyRule{Name: "x", Via: "office"}, "api.err.policy_empty"},
		{config.PolicyRule{Name: "x", Via: "office", Match: config.MatchSpec{IPRanges: []string{"10.0.0.0/99"}}}, "api.err.policy_bad_range"},
	}
	for _, tc := range cases {
		if got := key(ValidatePolicy(tc.rule, c)); got != tc.want {
			t.Errorf("%+v: key %q, want %q", tc.rule, got, tc.want)
		}
	}
}

func TestAddVPNProfile(t *testing.T) {
	keyring.MockInit()
	c, path := load(t)
	save := func(c *config.Config) error { _, err := Save(path, c); return err }
	name, typ, user, method := "work", "ciscoanyconnect", "ana", "credentials"
	host := "vpn.example.com"
	p := VPNProfile{Name: &name, Type: &typ, Host: &host, Username: &user, AuthMethod: &method, Password: "pw"}
	if _, err := AddVPNProfile(c, p, save); err != nil {
		t.Fatal(err)
	}
	if v := c.VPNs["work"]; v.Auth.PasswordKeychain != "work.password" || v.Auth.UsernameKeychain != "work.username" {
		t.Fatalf("profile = %+v", v)
	}
	if pw, _ := config.GetCredential("work", "password"); pw != "pw" {
		t.Fatalf("keychain = %q", pw)
	}
	if _, err := AddVPNProfile(c, p, save); key(err) != ErrKeyVPNExists {
		t.Fatalf("duplicate: %v", err)
	}
	empty := ""
	if _, err := AddVPNProfile(c, VPNProfile{Name: &empty}, save); key(err) != "api.err.name_required" {
		t.Fatalf("no name: %v", err)
	}
	// A failed save puts the keychain back as it was.
	name2 := "work2"
	p.Name = &name2
	if _, err := AddVPNProfile(c, p, func(*config.Config) error { return errors.New("disk full") }); err == nil {
		t.Fatal("save error lost")
	}
	if _, err := config.GetCredential("work2", "password"); err == nil {
		t.Fatal("password left in the keychain after a failed save")
	}
}

func TestAuthDefaults(t *testing.T) {
	if DefaultAuthMethod("wireguard") != "certificate" || DefaultAuthMethod("nordvpn") != "credentials" {
		t.Fatal("default auth method")
	}
	if !NeedsPassword("forticlient", "credentials") || NeedsPassword("openvpn", "certificate") || NeedsPassword("nordvpn", "credentials") {
		t.Fatal("needs password")
	}
}
