package privops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestPrepareWGAppRoutingKeepsPeerDefaults(t *testing.T) {
	data := []byte(dualStackConfig)
	got, err := prepareWGAppRouting(data, WGAppRouting{Mode: AppRouteFallback, Executables: []string{"/usr/bin/example"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"Table = off", "AllowedIPs = 0.0.0.0/0, ::/0"} {
		if !bytes.Contains(got, []byte(required)) {
			t.Errorf("missing %q", required)
		}
	}
	if bytes.Contains(got, []byte("DNS =")) {
		t.Error("app egress changes host DNS")
	}
	if string(data) != dualStackConfig {
		t.Error("source profile mutated")
	}
}

func TestWGAppRoutingLifecycle(t *testing.T) {
	const child = "KONGTROL_TEST_WG_APP_NETWORK"
	requireAppClassifierTest(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv(child) != "1" {
		cmd := exec.Command("unshare", "--net", "--", executable, "-test.run=^TestWGAppRoutingLifecycle$", "-test.v")
		cmd.Env = append(os.Environ(), child+"=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("isolated WireGuard test: %v\n%s", err, out)
		}
		return
	}
	ip := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("ip", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("ip %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	ip("link", "add", "ktappwg", "type", "wireguard")
	ip("addr", "add", "198.18.11.1/32", "dev", "ktappwg")
	ip("-6", "addr", "add", "fd00:11::1/128", "dev", "ktappwg", "nodad")
	ip("link", "set", "ktappwg", "up")
	local := NewLocal(LocalOptions{StateDir: t.TempDir()})
	route, err := local.startWGAppRouting("ktappwg", os.Getuid(), WGAppRouting{Mode: AppRouteFallback, Executables: []string{executable}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := route.close(); err != nil {
			t.Error(err)
		}
	})
	getMark := func(family, kind int) int {
		t.Helper()
		fd, err := unix.Socket(family, kind|unix.SOCK_CLOEXEC, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer unix.Close(fd)
		mark, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_MARK)
		if err != nil {
			t.Fatal(err)
		}
		return mark
	}
	for _, tc := range []struct {
		family       int
		arg, address string
	}{
		{unix.AF_INET, "-4", "198.18.22.5"},
		{unix.AF_INET6, "-6", "fd00:22::5"},
	} {
		for _, kind := range []int{unix.SOCK_STREAM, unix.SOCK_DGRAM} {
			mark := getMark(tc.family, kind)
			if mark == 0 {
				t.Fatal("socket was not marked")
			}
			if out := ip(tc.arg, "route", "get", tc.address, "mark", fmt.Sprint(mark)); !strings.Contains(out, "dev ktappwg ") {
				t.Fatalf("wrong destination interface: %s", out)
			}
		}
	}
	if err := route.close(); err != nil {
		t.Fatal(err)
	}
	if mark := getMark(unix.AF_INET, unix.SOCK_STREAM); mark != 0 {
		t.Fatalf("after cleanup mark = %d", mark)
	}
}
