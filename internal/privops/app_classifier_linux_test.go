package privops

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const appClassifierTestEnv = "KONGTROL_TEST_APP_CLASSIFIER"

func TestAppClassifierSocketMark(t *testing.T) {
	const (
		probeEnv = "KONGTROL_TEST_APP_SOCKET_PROBE"
		childEnv = "KONGTROL_TEST_APP_SOCKET_CHILD"
		mark     = 0x4B010001
	)
	if os.Getenv(probeEnv) != "" {
		fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer unix.Close(fd)
		value, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_MARK)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("mark:", value)
		if child := os.Getenv(childEnv); child != "" {
			cmd := exec.Command(child, "-test.run=^TestAppClassifierSocketMark$")
			cmd.Env = append(os.Environ(), childEnv+"=")
			out, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(out), "mark: 0\n") {
				t.Fatalf("unassigned child inherited mark: %v\n%s", err, out)
			}
		}
		return
	}
	classifier, cgroup, executable := newAppClassifierFixture(t)
	other := copyAppClassifierExecutable(t, executable)
	setAppClassifierMark(t, classifier, executable, mark)
	for _, tc := range []struct {
		executable string
		mark       int
	}{
		{executable, mark},
		{other, 0},
	} {
		cmd := exec.Command("sh", "-c", `echo "$$" > "$1/cgroup.procs" || exit; shift; exec "$@"`,
			"sh", cgroup, tc.executable, "-test.run=^TestAppClassifierSocketMark$")
		cmd.Env = append(os.Environ(), probeEnv+"=1")
		if tc.executable == executable {
			cmd.Env = append(cmd.Env, childEnv+"="+other)
		}
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), "mark: "+strconv.Itoa(tc.mark)+"\n") {
			t.Errorf("executable %q, want mark %#x: %v\n%s", tc.executable, tc.mark, err, out)
		}
	}
}

func TestAppClassifierSameDestination(t *testing.T) {
	const (
		modeEnv     = "KONGTROL_TEST_APP_ROUTE_MODE"
		destination = "198.18.10.10"
		payload     = "kongtrol same destination probe"
		firstMark   = 0x4B010001
		secondMark  = 0x4B010002
	)
	switch os.Getenv(modeEnv) {
	case "send":
		conn, err := net.Dial("udp4", destination+":59208")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if _, err := io.WriteString(conn, payload); err != nil {
			t.Fatal(err)
		}
		return
	case "disconnected":
		// The selected interface is gone, but a normal route to the same
		// address still exists. A marked socket must not use it.
		conn, err := net.Dial("udp4", destination+":59208")
		if !errors.Is(err, unix.EHOSTUNREACH) && !errors.Is(err, unix.ENETUNREACH) {
			if conn != nil {
				conn.Close()
			}
			t.Fatalf("expected unreachable destination, got %v", err)
		}
		return
	case "isolated":
	default:
		requireAppClassifierTest(t)
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("unshare", "--net", "--", executable, "-test.run=^TestAppClassifierSameDestination$", "-test.v")
		cmd.Env = append(os.Environ(), modeEnv+"=isolated")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("isolated network probe: %v\n%s", err, out)
		}
		t.Logf("%s", out)
		return
	}

	classifier, cgroup, first := newAppClassifierFixture(t)
	second := copyAppClassifierExecutable(t, first)
	third := copyAppClassifierExecutable(t, first)
	ip := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("ip", args...).CombinedOutput(); err != nil {
			t.Fatalf("ip %v: %v\n%s", args, err, out)
		}
	}
	for _, target := range []struct {
		iface, peer, source string
		table, priority     int
		mark                uint32
	}{
		{"testvpn1", "testpeer1", "198.18.11.1", 101, 101, firstMark},
		{"testvpn2", "testpeer2", "198.18.12.1", 102, 102, secondMark},
		{"testdirect", "testpeer3", "198.18.13.1", 0, 0, 0},
	} {
		ip("link", "add", target.iface, "type", "veth", "peer", "name", target.peer)
		ip("link", "set", target.iface, "up")
		ip("link", "set", target.peer, "up")
		ip("addr", "add", target.source+"/32", "dev", target.iface)
		peer, err := net.InterfaceByName(target.peer)
		if err != nil {
			t.Fatal(err)
		}
		ip("neigh", "add", destination, "lladdr", peer.HardwareAddr.String(), "nud", "permanent", "dev", target.iface)
		if target.mark == 0 {
			ip("route", "add", destination+"/32", "dev", target.iface, "src", target.source)
			continue
		}
		egress, err := netlink.LinkByName(target.iface)
		if err != nil {
			t.Fatal(err)
		}
		route, err := openAppRouteTable(egress, netip.MustParsePrefix(destination+"/32"), target.table, target.priority, target.mark)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := route.Close(); err != nil {
				t.Errorf("remove app egress: %v", err)
			}
		})
	}
	setAppClassifierMark(t, classifier, first, firstMark)
	setAppClassifierMark(t, classifier, second, secondMark)
	send := func(executable, mode string) {
		t.Helper()
		cmd := exec.Command("sh", "-c", `echo "$$" > "$1/cgroup.procs" || exit; shift; exec "$@"`,
			"sh", cgroup, executable, "-test.run=^TestAppClassifierSameDestination$")
		cmd.Env = append(os.Environ(), modeEnv+"="+mode)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s from %s: %v\n%s", mode, executable, err, out)
		}
	}
	for _, tc := range []struct {
		executable, peer string
	}{
		{first, "testpeer1"},
		{second, "testpeer2"},
		{third, "testpeer3"},
	} {
		peer, err := net.InterfaceByName(tc.peer)
		if err != nil {
			t.Fatal(err)
		}
		// AF_PACKET takes the EtherType in network byte order.
		var ethertype [2]byte
		binary.BigEndian.PutUint16(ethertype[:], unix.ETH_P_IP)
		proto := binary.NativeEndian.Uint16(ethertype[:])
		fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, int(proto))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { unix.Close(fd) })
		if err := unix.Bind(fd, &unix.SockaddrLinklayer{Protocol: proto, Ifindex: peer.Index}); err != nil {
			t.Fatal(err)
		}
		if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &unix.Timeval{Sec: 2}); err != nil {
			t.Fatal(err)
		}
		send(tc.executable, "send")
		var packet [2048]byte
		n, _, err := unix.Recvfrom(fd, packet[:], 0)
		if err != nil || !bytes.HasSuffix(packet[:n], []byte(payload)) {
			t.Fatalf("expected test packet on %s: %v", tc.peer, err)
		}
		t.Logf("observed packet to %s:59208 on %s", destination, tc.peer)
	}
	ip("link", "delete", "testvpn1")
	send(first, "disconnected")
	send(second, "send")
	send(third, "send")
}

func requireAppClassifierTest(t *testing.T) {
	t.Helper()
	if os.Getenv(appClassifierTestEnv) != "1" {
		t.Skipf("set %s=1 to run with a writable cgroup v2 hierarchy and BPF capabilities", appClassifierTestEnv)
	}
}

func newAppClassifierFixture(t *testing.T) (*appSocketClassifier, string, string) {
	t.Helper()
	requireAppClassifierTest(t)
	cgroup, err := os.MkdirTemp("/sys/fs/cgroup", "kongtrol-test-app-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(cgroup); err != nil {
			t.Errorf("remove test cgroup: %v", err)
		}
	})
	classifier, err := openAppSocketClassifier(cgroup)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := classifier.Close(); err != nil {
			t.Errorf("close test classifier: %v", err)
		}
	})
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return classifier, cgroup, executable
}

func setAppClassifierMark(t *testing.T, classifier *appSocketClassifier, path string, mark uint32) {
	t.Helper()
	key, err := executableSocketKey(path, uint32(os.Getuid()))
	if err != nil {
		t.Fatal(err)
	}
	if err := classifier.marks.Put(key, mark); err != nil {
		t.Fatal(err)
	}
}

func copyAppClassifierExecutable(t *testing.T, executable string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), "probe")
	src, err := os.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(dst, src)
	if err := errors.Join(copyErr, dst.Close()); err != nil {
		t.Fatal(err)
	}
	return target
}
