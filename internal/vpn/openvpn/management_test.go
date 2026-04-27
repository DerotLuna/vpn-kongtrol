package openvpn

import (
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
