package appattach

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

var (
	testMatcher  = Matcher{Patterns: []string{"surfshark*"}}
	testProvider = Provider{App: App{DisplayName: "Surfshark"}, Matcher: testMatcher}
)

func phaseOf(s *Session) vpn.AttachPhase {
	if st := s.AttachState(); st != nil {
		return st.Phase
	}
	return ""
}

func TestConnectAlreadyUpAttaches(t *testing.T) {
	f := &fakeNet{up: true}
	launched := 0
	s := newTestSession(f, &launched)
	var phases []vpn.AttachPhase
	hook := func(a vpn.AttachState) { phases = append(phases, a.Phase) }
	if err := s.Connect(context.Background(), testProvider, Options{AutoConnect: true, WaitTimeout: time.Second}, hook); err != nil {
		t.Fatal(err)
	}
	if launched != 0 {
		t.Errorf("app launched %d times, want 0", launched)
	}
	if len(phases) != 1 || phases[0] != vpn.PhaseAttached {
		t.Errorf("phases = %v, want [attached]", phases)
	}
	if st := s.Status(testProvider, true); st != vpn.StatusConnected {
		t.Errorf("status = %s", st)
	}
	if info := s.Info(); info == nil || info.InterfaceName != "surfshark_wg" {
		t.Errorf("info = %+v", info)
	}
}

func TestConnectAutomationSucceeds(t *testing.T) {
	f := &fakeNet{}
	launched := 0
	s := newTestSession(f, &launched)
	p := Provider{
		App: App{DisplayName: "NordVPN"}, Matcher: testMatcher, Location: "United States",
		AutoConnect: func(context.Context) error { f.set(true); return nil },
	}
	if err := s.Connect(context.Background(), p, Options{AutoConnect: true, WaitTimeout: time.Second}, nil); err != nil {
		t.Fatal(err)
	}
	if launched != 0 {
		t.Errorf("app launched %d times, want 0", launched)
	}
	st := s.AttachState()
	if st == nil || st.Phase != vpn.PhaseAuto || st.Location != "United States" {
		t.Errorf("state = %+v", st)
	}
}

func TestConnectAutomationFailsFallsBackToUser(t *testing.T) {
	f := &fakeNet{}
	launched := 0
	s := newTestSession(f, &launched)
	p := Provider{
		App: App{DisplayName: "FortiClient"}, Matcher: testMatcher,
		AutoConnect: func(context.Context) error { return errors.New("gui layout changed\nstack trace") },
	}
	var phases []vpn.AttachState
	hook := func(a vpn.AttachState) {
		phases = append(phases, a)
		if a.Phase != vpn.PhaseWaitingUser {
			return
		}
		if s.Status(p, false) != vpn.StatusConnecting {
			t.Error("status during wait should be connecting")
		}
		go func() { time.Sleep(20 * time.Millisecond); f.set(true) }()
	}
	if err := s.Connect(context.Background(), p, Options{AutoConnect: true, WaitTimeout: 2 * time.Second}, hook); err != nil {
		t.Fatal(err)
	}
	if len(phases) != 2 || phases[0].Phase != vpn.PhaseWaitingUser || phases[1].Phase != vpn.PhaseUser {
		t.Fatalf("phases = %+v, want waiting_user then user", phases)
	}
	if phases[0].Reason != "gui layout changed" || phases[0].TimeoutSeconds != 2 {
		t.Errorf("waiting state = %+v", phases[0])
	}
	if launched != 1 {
		t.Errorf("app launched %d times, want 1", launched)
	}
}

func TestConnectAutoConnectDisabledSkipsAutomation(t *testing.T) {
	f := &fakeNet{}
	launched := 0
	s := newTestSession(f, &launched)
	called := false
	p := Provider{
		App: App{DisplayName: "NordVPN"}, Matcher: testMatcher,
		AutoConnect: func(context.Context) error { called = true; return nil },
	}
	err := s.Connect(context.Background(), p, Options{AutoConnect: false, WaitTimeout: 30 * time.Millisecond}, nil)
	if !errors.Is(err, vpn.ErrUserActionTimeout) {
		t.Fatalf("err = %v, want ErrUserActionTimeout", err)
	}
	if called {
		t.Error("automation ran with auto_connect false")
	}
	if !p.ManualReconnect(Options{AutoConnect: false}) || p.ManualReconnect(Options{AutoConnect: true}) {
		t.Error("ManualReconnect mismatch")
	}
	if st := s.Status(p, true); st != vpn.StatusDisconnected || phaseOf(s) != "" {
		t.Errorf("after timeout: status %s phase %q", st, phaseOf(s))
	}
}

func TestConnectCancelled(t *testing.T) {
	s := newTestSession(&fakeNet{}, new(int))
	ctx, cancel := context.WithCancel(context.Background())
	err := s.Connect(ctx, testProvider, Options{WaitTimeout: time.Minute}, func(vpn.AttachState) { cancel() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestDisconnect(t *testing.T) {
	t.Run("cli", func(t *testing.T) {
		f := &fakeNet{up: true}
		s := newTestSession(f, new(int))
		p := Provider{App: App{DisplayName: "NordVPN"}, Matcher: testMatcher,
			Disconnect: func(context.Context) error { f.set(false); return nil }}
		_ = s.Status(p, false)
		if err := s.Disconnect(context.Background(), p, Options{}); err != nil {
			t.Fatal(err)
		}
		if st := s.Status(p, false); st != vpn.StatusDisconnected || phaseOf(s) != "" {
			t.Errorf("status = %s phase %q", st, phaseOf(s))
		}
	})
	t.Run("manual", func(t *testing.T) {
		f := &fakeNet{up: true}
		s := newTestSession(f, new(int))
		_ = s.Status(testProvider, true)
		err := s.Disconnect(context.Background(), testProvider, Options{})
		if !errors.Is(err, vpn.ErrManualDisconnect) {
			t.Fatalf("err = %v", err)
		}
		if st := s.Status(testProvider, true); st != vpn.StatusConnected || phaseOf(s) != vpn.PhaseWaitingDisconnect {
			t.Errorf("before user disconnects: %s / %q", st, phaseOf(s))
		}
		f.set(false)
		if st := s.Status(testProvider, true); st != vpn.StatusDisconnected || phaseOf(s) != "" {
			t.Errorf("after requested down: %s / %q, want disconnected without dropped phase", st, phaseOf(s))
		}
	})
	t.Run("not up", func(t *testing.T) {
		s := newTestSession(&fakeNet{}, new(int))
		if err := s.Disconnect(context.Background(), testProvider, Options{}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestStatusPhases(t *testing.T) {
	f := &fakeNet{}
	s := newTestSession(f, new(int))
	if st := s.Status(testProvider, true); st != vpn.StatusDisconnected || phaseOf(s) != "" {
		t.Fatalf("zero: %q / %q", st, phaseOf(s))
	}
	f.set(true) // user connected in the app, outside Kongtrol
	if st := s.Status(testProvider, true); st != vpn.StatusConnected || phaseOf(s) != vpn.PhaseAttached {
		t.Fatalf("external up: %q / %q", st, phaseOf(s))
	}
	f.set(false) // dropped, only the user can reconnect
	if st := s.Status(testProvider, true); st != vpn.StatusDisconnected || phaseOf(s) != vpn.PhaseDropped {
		t.Fatalf("manual drop: %q / %q", st, phaseOf(s))
	}
	f.set(true) // back up: re-attached
	_ = s.Status(testProvider, true)
	if phaseOf(s) != vpn.PhaseAttached {
		t.Fatalf("re-attach: %q", phaseOf(s))
	}
	f.set(false) // drop with an automatic reconnect path: no dropped phase
	_ = s.Status(testProvider, false)
	if phaseOf(s) != "" {
		t.Fatalf("auto drop: %q", phaseOf(s))
	}
}

func TestShortReason(t *testing.T) {
	long := errors.New(strings.Repeat("x", 100))
	cases := []struct {
		err  error
		want string
	}{
		{nil, ""},
		{errors.New("not logged in\nmore"), "not logged in"},
		{errWaitTimeout, "timeout"},
		{long, strings.Repeat("x", maxReasonLen-1) + "…"},
	}
	for _, c := range cases {
		if got := ShortReason(c.err); got != c.want {
			t.Errorf("ShortReason(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}

func TestMessageKey(t *testing.T) {
	cases := []struct {
		st   vpn.AttachState
		want string
	}{
		{vpn.AttachState{Phase: vpn.PhaseAttached}, "attach.attached"},
		{vpn.AttachState{Phase: vpn.PhaseAuto}, "attach.auto"},
		{vpn.AttachState{Phase: vpn.PhaseAuto, Location: "US"}, "attach.auto_location"},
		{vpn.AttachState{Phase: vpn.PhaseUser}, "attach.user"},
		{vpn.AttachState{Phase: vpn.PhaseWaitingUser}, "attach.waiting_user"},
		{vpn.AttachState{Phase: vpn.PhaseWaitingUser, Reason: "x"}, "attach.automation_failed"},
		{vpn.AttachState{Phase: vpn.PhaseWaitingDisconnect}, "attach.waiting_disconnect"},
		{vpn.AttachState{Phase: vpn.PhaseDropped}, "attach.dropped"},
		{vpn.AttachState{}, ""},
	}
	for _, c := range cases {
		if got, _ := c.st.MessageKey(); got != c.want {
			t.Errorf("MessageKey(%+v) = %q, want %q", c.st, got, c.want)
		}
	}
}
