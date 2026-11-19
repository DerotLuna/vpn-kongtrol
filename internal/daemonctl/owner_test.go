package daemonctl

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/sockowner"
)

// tokenServer records every token it receives.
func tokenServer(t *testing.T, want *atomic.Value) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var got atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Add(1)
		if r.Header.Get(TokenHeader) != want.Load().(string) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func fakeOwner(t *testing.T, res sockowner.Result, err error, pid uint32, pidOK bool) {
	t.Helper()
	oldC, oldP, oldKA := checkOwner, daemonPID, transport.DisableKeepAlives
	t.Cleanup(func() {
		checkOwner, daemonPID = oldC, oldP
		transport.DisableKeepAlives = oldKA
		transport.CloseIdleConnections()
	})
	// Every request must dial (and so run the owner check) anew: a kept-alive
	// connection from the previous case can return to the pool after
	// CloseIdleConnections and be reused without a new check (flaky on CI).
	transport.DisableKeepAlives = true
	checkOwner = func(netip.AddrPort, netip.AddrPort) (sockowner.Result, error) { return res, err }
	daemonPID = func() (uint32, time.Time, bool) { return pid, pidWritten, pidOK }
	transport.CloseIdleConnections() // every case dials anew
}

// pidWritten is when the fake PID file was written; before is earlier.
var (
	pidWritten = time.Unix(1_700_000_000, 0)
	before     = pidWritten.Add(-time.Minute)
)

// elevated is a Denied owner whose process opened but whose token did not.
func elevated(pid uint32, created time.Time) sockowner.Result {
	return sockowner.Result{Verdict: sockowner.Denied, PID: pid, ProcessOpened: true, Created: created}
}

// The token never reaches a listener another user owns (ADR-0062).
func TestClientRefusesForeignListener(t *testing.T) {
	var want atomic.Value
	want.Store("tok")
	srv, hits := tokenServer(t, &want)
	cases := []struct {
		name   string
		res    sockowner.Result
		err    error
		pid    uint32
		pidOK  bool
		refuse bool
	}{
		{"same user", sockowner.Result{Verdict: sockowner.Self}, nil, 0, false, false},
		{"another user", sockowner.Result{Verdict: sockowner.Other}, nil, 0, false, true},
		{"not inspectable, not our daemon", elevated(77, before), nil, 12, true, true},
		{"not inspectable, no PID file", elevated(77, before), nil, 0, false, true},
		{"our elevated daemon (PID file)", elevated(77, before), nil, 77, true, false},
		// Another user's process cannot even be opened: never ours, even
		// when a stale PID file names its (reused) PID.
		{"process not openable, PID matches", sockowner.Result{Verdict: sockowner.Denied, PID: 77}, nil, 77, true, true},
		// Stale PID file (crash, reboot): the PID now belongs to a process
		// created after the file was written.
		{"stale PID file, newer process", elevated(77, pidWritten.Add(time.Minute)), nil, 77, true, true},
		{"opened but no creation time", sockowner.Result{Verdict: sockowner.Denied, PID: 77, ProcessOpened: true}, nil, 77, true, true},
		{"unknown (macOS) proceeds", sockowner.Result{}, sockowner.ErrUnsupported, 0, false, false},
		{"lookup failed proceeds", sockowner.Result{}, errors.New("table"), 0, false, false},
	}
	for _, tc := range cases {
		fakeOwner(t, tc.res, tc.err, tc.pid, tc.pidOK)
		before := hits.Load()
		ok := New(srv.URL, "tok").Probe()
		var fe *ForeignListenerError
		_, err := New(srv.URL, "tok").Statuses()
		refused := errors.As(err, &fe)
		if refused != tc.refuse || ok == tc.refuse {
			t.Errorf("%s: probe=%v err=%v, want refuse=%v", tc.name, ok, err, tc.refuse)
		}
		if tc.refuse && hits.Load() != before {
			t.Errorf("%s: the listener received a request", tc.name)
		}
	}
}

func TestForeignListenerErrorIsGuided(t *testing.T) {
	e := &ForeignListenerError{Addr: "127.0.0.1:9741", Lang: i18n.ES}
	if !strings.Contains(e.Error(), "9741") || !strings.Contains(e.Error(), "Qué hacer") {
		t.Fatalf("es: %q", e.Error())
	}
	if !strings.Contains(e.Localized(i18n.EN), "What to do") {
		t.Fatalf("en: %q", e.Localized(i18n.EN))
	}
}

// The tray and the CLI keep working across a daemon restart that rotates
// the token: TokenSource is read per request.
func TestClientFollowsRotatedToken(t *testing.T) {
	fakeOwner(t, sockowner.Result{Verdict: sockowner.Self}, nil, 0, false)
	var want atomic.Value
	want.Store("first")
	srv, _ := tokenServer(t, &want)
	var file atomic.Value
	file.Store("first")
	c := New(srv.URL, "first")
	c.TokenSource = func() (string, error) { return file.Load().(string), nil }
	if _, err := c.Statuses(); err != nil {
		t.Fatal(err)
	}
	want.Store("second") // the daemon restarted
	file.Store("second")
	if _, err := c.Statuses(); err != nil {
		t.Fatalf("after rotation: %v", err)
	}
	stale := New(srv.URL, "first")
	if _, err := stale.Statuses(); err == nil {
		t.Fatal("an old token must stop working")
	}
}

// Ensure explains a taken port instead of spawning a daemon that cannot bind.
func TestSpawnerRefusesForeignPort(t *testing.T) {
	old := checkListenerFn
	t.Cleanup(func() { checkListenerFn = old })
	checkListenerFn = func(string) error { return &ForeignListenerError{Addr: "127.0.0.1:9741"} }
	sp := Spawner{
		Probe: func() bool { return false },
		Spawn: func(string) (<-chan error, error) {
			t.Fatal("must not spawn")
			return nil, nil
		},
		Base:     "http://127.0.0.1:9741",
		Lang:     i18n.EN,
		LockPath: t.TempDir() + "/spawn.lock",
	}
	_, err := sp.Ensure(context.Background())
	var fe *ForeignListenerError
	if !errors.As(err, &fe) || !strings.Contains(err.Error(), "What to do") {
		t.Fatalf("err=%v", err)
	}
}
