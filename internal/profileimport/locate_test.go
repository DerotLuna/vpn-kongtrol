package profileimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const tlsKey = "-----BEGIN OpenVPN Static key V1-----\nab\n-----END OpenVPN Static key V1-----\n"

const ovpnOwner = "client\ndev tun\nremote vpn.example.com 1194\nca ca.crt\ntls-crypt ta.key\n"

// realTemp is t.TempDir with its links resolved (macOS /var → /private/var):
// the outside-folder checks refuse a path through a link.
func realTemp(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// withRefs writes the config and the files it names in dir.
func withRefs(t *testing.T, dir, conf string) {
	t.Helper()
	write(t, filepath.Join(dir, "rluna.conf"), conf)
	write(t, filepath.Join(dir, "ca.crt"), caPEM)
	write(t, filepath.Join(dir, "ta.key"), tlsKey)
}

func TestLocate_OneIdenticalCopy(t *testing.T) {
	root := realTemp(t)
	want := filepath.Join(root, "OpenVPN", "config")
	withRefs(t, want, ovpnOwner)
	// Same name, same size, other bytes: never a match.
	write(t, filepath.Join(root, "Downloads", "rluna.conf"), strings.Replace(ovpnOwner, "example", "exbmple", 1))
	l := &Locator{Names: []string{"rluna.conf"}, Roots: []string{filepath.Join(root, "Downloads"), filepath.Join(root, "OpenVPN")}}
	if got := l.Find([]byte(ovpnOwner)); got != want {
		t.Fatalf("Find = %q, want %q", got, want)
	}
}

func TestLocate_NoneAndNoName(t *testing.T) {
	root := realTemp(t)
	withRefs(t, filepath.Join(root, "a"), ovpnOwner)
	if got := (&Locator{Names: []string{"other.conf"}, Roots: []string{root}}).Find([]byte(ovpnOwner)); got != "" {
		t.Fatalf("other name found %q", got)
	}
	if got := (&Locator{Names: []string{"rluna.conf"}, Roots: []string{root}}).Find([]byte(ovpnOwner + "#")); got != "" {
		t.Fatalf("other bytes found %q", got)
	}
	if got := (&Locator{Roots: []string{root}}).Find([]byte(ovpnOwner)); got != "" {
		t.Fatalf("no name found %q", got)
	}
	if got := (&Locator{Names: []string{"rluna.conf"}, Roots: []string{root}, MaxDepth: 0}).Find([]byte(ovpnOwner)); got == "" {
		t.Fatal("depth 1 not searched")
	}
	if got := (&Locator{Names: []string{"rluna.conf"}, Roots: []string{"relative/dir", `\\server\share`}}).Find([]byte(ovpnOwner)); got != "" {
		t.Fatalf("non-local root searched: %q", got)
	}
}

func TestLocate_SeveralCopies(t *testing.T) {
	root := realTemp(t)
	withRefs(t, filepath.Join(root, "a"), ovpnOwner)
	write(t, filepath.Join(root, "b", "rluna.conf"), ovpnOwner) // the copy alone
	l := &Locator{Names: []string{"rluna.conf"}, Roots: []string{root}}
	if got := l.Find([]byte(ovpnOwner)); got != filepath.Join(root, "a") {
		t.Fatalf("Find = %q, want the folder with every file", got)
	}
	withRefs(t, filepath.Join(root, "c"), ovpnOwner) // a second complete folder
	if got := l.Find([]byte(ovpnOwner)); got != "" {
		t.Fatalf("ambiguous copies guessed %q", got)
	}
}

func TestLocate_LinksAreNotFollowed(t *testing.T) {
	root := realTemp(t)
	other := realTemp(t)
	withRefs(t, filepath.Join(other, "vpn"), ovpnOwner)
	if err := os.Symlink(filepath.Join(other, "vpn"), filepath.Join(root, "link")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if got := (&Locator{Names: []string{"rluna.conf"}, Roots: []string{root}}).Find([]byte(ovpnOwner)); got != "" {
		t.Fatalf("followed a link: %q", got)
	}
	if got := (&Locator{Names: []string{"rluna.conf"}, Roots: []string{filepath.Join(root, "link")}}).Find([]byte(ovpnOwner)); got != "" {
		t.Fatalf("a root that is a link was searched: %q", got)
	}
}

func TestLocate_Bounds(t *testing.T) {
	root := realTemp(t)
	for i := range 20 {
		write(t, filepath.Join(root, "f"+string(rune('a'+i))+".txt"), "x") // beside z: File.ReadDir order is not sorted
	}
	withRefs(t, filepath.Join(root, "z"), ovpnOwner)
	if got := (&Locator{Names: []string{"rluna.conf"}, Roots: []string{root}, MaxEntries: 5}).Find([]byte(ovpnOwner)); got != "" {
		t.Fatalf("entry bound ignored: %q", got)
	}
	if got := (&Locator{Names: []string{"rluna.conf"}, Roots: []string{root}, Budget: time.Nanosecond}).Find([]byte(ovpnOwner)); got != "" {
		t.Fatalf("time budget ignored: %q", got)
	}
	deep := filepath.Join(root, "d1", "d2", "d3", "d4")
	withRefs(t, deep, ovpnOwner+"# deep\n")
	if got := (&Locator{Names: []string{"rluna.conf"}, Roots: []string{root}}).Find([]byte(ovpnOwner + "# deep\n")); got != "" {
		t.Fatalf("searched deeper than 3: %q", got)
	}
}

// A dropped file (no folder) whose original sits next to its certificates
// imports in one go, and the result says where they were found.
func TestImport_LocatesTheOriginalOfADroppedFile(t *testing.T) {
	root := realTemp(t)
	src := filepath.Join(root, "OpenVPN", "config")
	withRefs(t, src, ovpnOwner)
	refs, err := NewRefs("", nil)
	if err != nil {
		t.Fatal(err)
	}
	refs.Disk = true
	refs.Locate = &Locator{Names: []string{"rluna.conf"}, Roots: []string{root}}
	res, err := Import(t.TempDir(), Request{Name: "rluna", Type: "openvpn", Content: []byte(ovpnOwner), Refs: refs})
	if err != nil {
		t.Fatal(err)
	}
	if res.FoundIn != src || len(res.Included) != 2 {
		t.Fatalf("%+v", res)
	}
	// Not found: the missing files are listed as before.
	refs.Locate = &Locator{Names: []string{"rluna.conf"}, Roots: []string{t.TempDir()}}
	if _, err := Import(t.TempDir(), Request{Name: "rluna", Type: "openvpn", Content: []byte(ovpnOwner), Refs: refs}); key(t, err) != ErrKeyMissingFiles {
		t.Fatal(err)
	}
}
