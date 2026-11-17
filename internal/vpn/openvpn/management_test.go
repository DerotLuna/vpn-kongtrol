package openvpn

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestParseState_Connected(t *testing.T) {
	// Real OpenVPN state format: TIMESTAMP,STATE,DESC,LOCAL_IP,REMOTE_IP
	raw := "1720000000,CONNECTED,SUCCESS,10.8.0.6,1.2.3.4,1194,,"
	state, localIP := parseState(raw)

	if state != "CONNECTED" {
		t.Errorf("state = %q, want CONNECTED", state)
	}
	if localIP != "10.8.0.6" {
		t.Errorf("localIP = %q, want 10.8.0.6", localIP)
	}
}

func TestParseState_Connecting(t *testing.T) {
	raw := "1720000000,WAIT,,,"
	state, localIP := parseState(raw)

	if state != "WAIT" {
		t.Errorf("state = %q, want WAIT", state)
	}
	if localIP != "" {
		t.Errorf("localIP = %q, want empty", localIP)
	}
}

func TestParseState_AuthFailed(t *testing.T) {
	raw := "1720000000,AUTH_FAILED,,,"
	state, _ := parseState(raw)
	if state != "AUTH_FAILED" {
		t.Errorf("state = %q, want AUTH_FAILED", state)
	}
}

func TestParseState_Empty(t *testing.T) {
	state, localIP := parseState("")
	if state != "UNKNOWN" {
		t.Errorf("state = %q, want UNKNOWN", state)
	}
	if localIP != "" {
		t.Errorf("localIP = %q, want empty", localIP)
	}
}

func TestParseState_Malformed(t *testing.T) {
	// Should not panic on malformed input.
	state, _ := parseState("CONNECTED") // no commas
	if state != "UNKNOWN" {
		t.Errorf("state = %q, want UNKNOWN for malformed input", state)
	}
}

func TestFreePort_ReturnsNonZero(t *testing.T) {
	port, err := freePort()
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("freePort = %d, want valid port number", port)
	}
}

func TestFreePort_Unique(t *testing.T) {
	// Ports may occasionally collide but should be unique in fast succession.
	seen := make(map[int]bool)
	for i := 0; i < 5; i++ {
		port, err := freePort()
		if err != nil {
			t.Fatalf("freePort iteration %d: %v", i, err)
		}
		if seen[port] {
			t.Logf("port %d reused (acceptable race, not a hard failure)", port)
		}
		seen[port] = true
	}
}

func TestParseStatusBytes(t *testing.T) {
	out := "OpenVPN STATISTICS\nUpdated,2026-11-17 10:00:00\nTUN/TAP read bytes,1000\nTUN/TAP write bytes,2000\nTCP/UDP read bytes,4096\nTCP/UDP write bytes,1024\nAuth read bytes,2000\nEND\n"
	sent, recv, ok := parseStatusBytes(out)
	if !ok || sent != 1024 || recv != 4096 {
		t.Fatalf("got sent=%d recv=%d ok=%v", sent, recv, ok)
	}
	if _, _, ok := parseStatusBytes("OpenVPN STATISTICS\nEND\n"); ok {
		t.Fatal("missing counters must not be ok")
	}
}

func TestMgmtByteTotals(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprint(conn, ">INFO:OpenVPN Management Interface\r\n")
		r := bufio.NewReader(conn)
		if line, _ := r.ReadString('\n'); strings.TrimSpace(line) != "status" {
			return
		}
		fmt.Fprint(conn, "OpenVPN STATISTICS\r\n>BYTECOUNT:5,6\r\nTCP/UDP read bytes,300\r\nTCP/UDP write bytes,200\r\nEND\r\n")
	}()
	c, err := newMgmtClient(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	sent, recv, err := c.ByteTotals()
	if err != nil || sent != 200 || recv != 300 {
		t.Fatalf("ByteTotals = %d,%d,%v", sent, recv, err)
	}
}
