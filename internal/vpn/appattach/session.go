package appattach

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

const (
	// DefaultWaitTimeout bounds step 3 (waiting for the user to connect in
	// the app) when the profile sets no wait_timeout. Generous on purpose:
	// the user may have to sign in or pass MFA first.
	DefaultWaitTimeout = 5 * time.Minute
	// MinWaitTimeout is the smallest accepted wait_timeout.
	MinWaitTimeout = 10 * time.Second

	defaultAutoWait       = 45 * time.Second
	defaultDisconnectWait = 10 * time.Second
	defaultPoll           = time.Second
	maxReasonLen          = 60
)

// Options are the profile's standard attach-mode settings, carried in
// vpn.AdapterConfig.Extra (see ParseOptions).
type Options struct {
	AutoConnect bool          // try CLI / GUI automation first (default true)
	WaitTimeout time.Duration // step-3 wait for the user (default DefaultWaitTimeout)
	Interface   string        // `interface:` override (comma-separated globs)
	AppPath     string        // `app_path:` override of the app to launch
}

// ParseOptions reads the attach-mode keys of AdapterConfig.Extra:
// auto_connect ("false" disables; anything else, including empty, keeps the
// default true), wait_timeout (Go duration), interface, app_path.
func ParseOptions(extra map[string]string) Options {
	o := Options{AutoConnect: true, WaitTimeout: DefaultWaitTimeout}
	if v, err := strconv.ParseBool(strings.TrimSpace(extra["auto_connect"])); err == nil {
		o.AutoConnect = v
	}
	if d, err := time.ParseDuration(strings.TrimSpace(extra["wait_timeout"])); err == nil && d >= MinWaitTimeout {
		o.WaitTimeout = d
	}
	o.Interface = strings.TrimSpace(extra["interface"])
	o.AppPath = strings.TrimSpace(extra["app_path"])
	return o
}

// Provider is what an adapter supplies for one Connect / Disconnect.
type Provider struct {
	App     App
	Matcher Matcher // already combined with the `interface:` override
	// Location is the requested target for messages ("United States"),
	// empty when the app picks.
	Location string
	// AutoConnect starts the connection through the official CLI or (for
	// corporate clients only) GUI automation; nil when this OS has none.
	// It may return before the tunnel is up. Best effort: an error is not
	// fatal, Connect falls back to waiting for the user.
	AutoConnect func(ctx context.Context) error
	// AutoWait bounds the wait for the interface after AutoConnect
	// succeeded (default 45s) before falling back to the user.
	AutoWait time.Duration
	// Disconnect tears the tunnel down through the CLI / automation; nil
	// when there is none (the user is asked to disconnect in the app).
	Disconnect func(ctx context.Context) error
	// DisconnectWait bounds the wait for the interface to drop after
	// Disconnect returned nil (default 10s).
	DisconnectWait time.Duration
}

// ManualReconnect reports whether only the user can bring the tunnel up
// (vpn.Capabilities.ManualReconnect).
func (p Provider) ManualReconnect(o Options) bool {
	return !o.AutoConnect || p.AutoConnect == nil
}

// Session is the attach-mode state an adapter embeds. All methods are
// safe for concurrent use; no lock is held while waiting, so Status()
// stays responsive during a long Connect.
type Session struct {
	mu          sync.Mutex
	status      vpn.Status
	tunnel      Tunnel
	connectedAt time.Time
	state       *vpn.AttachState
	cancelWait  context.CancelFunc

	// Test seams; nil means the real OS implementation.
	ifaces  func() []Iface
	service func([]string) (Tunnel, bool)
	launch  func(App, string) error
	poll    time.Duration
}

func (s *Session) detect(m Matcher) (Tunnel, bool) {
	list, svc := s.ifaces, s.service
	if list == nil {
		list = osInterfaces
	}
	if svc == nil {
		svc = osServiceTunnel
	}
	if t, ok := Find(list(), m); ok {
		return t, true
	}
	if len(m.Services) > 0 {
		return svc(m.Services)
	}
	return Tunnel{}, false
}

func (s *Session) doLaunch(app App, override string) error {
	if s.launch != nil {
		return s.launch(app, override)
	}
	return launchFunc(app, override)
}

func (s *Session) pollEvery() time.Duration {
	if s.poll > 0 {
		return s.poll
	}
	return defaultPoll
}

// Detect reports the provider's tunnel without changing session state.
func (s *Session) Detect(m Matcher) (Tunnel, bool) { return s.detect(m) }

// setState records a new phase and notifies hook (outside the lock).
func (s *Session) setState(st vpn.AttachState, hook func(vpn.AttachState)) {
	if st.Since.IsZero() {
		st.Since = time.Now()
	}
	s.mu.Lock()
	s.state = &st
	s.mu.Unlock()
	if hook != nil {
		hook(st)
	}
}

// attach records t as the connected tunnel reached through phase.
func (s *Session) attach(t Tunnel, phase vpn.AttachPhase, p Provider, hook func(vpn.AttachState)) {
	s.mu.Lock()
	s.attachLocked(t)
	s.mu.Unlock()
	s.setState(vpn.AttachState{Phase: phase, App: p.App.DisplayName, Location: p.Location}, hook)
}

// attachLocked records t as the connected tunnel. Caller holds s.mu.
func (s *Session) attachLocked(t Tunnel) {
	if s.status != vpn.StatusConnected || s.connectedAt.IsZero() {
		s.connectedAt = time.Now()
	}
	s.tunnel = t
	s.status = vpn.StatusConnected
}

// ShortReason condenses an automation error into one short line.
func ShortReason(err error) string {
	if err == nil {
		return ""
	}
	r := strings.TrimSpace(err.Error())
	if i := strings.IndexAny(r, "\r\n"); i >= 0 {
		r = strings.TrimSpace(r[:i])
	}
	if errors.Is(err, errWaitTimeout) {
		r = "timeout"
	}
	if runes := []rune(r); len(runes) > maxReasonLen {
		r = string(runes[:maxReasonLen-1]) + "…"
	}
	return r
}

// Connect runs the attach-mode connect flow (see the package doc). hook,
// when non-nil, is called on every phase change.
func (s *Session) Connect(ctx context.Context, p Provider, o Options, hook func(vpn.AttachState)) (err error) {
	s.mu.Lock()
	if s.status == vpn.StatusConnecting {
		s.mu.Unlock()
		return fmt.Errorf("%s: already connecting", p.App.DisplayName)
	}
	s.status = vpn.StatusConnecting
	s.state = nil
	ctx, cancel := context.WithCancel(ctx)
	s.cancelWait = cancel
	s.mu.Unlock()

	defer func() {
		cancel()
		s.mu.Lock()
		s.cancelWait = nil
		if err != nil {
			s.state = nil
			if s.status == vpn.StatusConnecting {
				s.status = vpn.StatusDisconnected
			}
		}
		s.mu.Unlock()
	}()

	// (1) Already up → attach.
	if t, ok := s.detect(p.Matcher); ok {
		s.attach(t, vpn.PhaseAttached, p, hook)
		return nil
	}

	// (2) Best-effort automation.
	var autoErr error
	if o.AutoConnect && p.AutoConnect != nil {
		if autoErr = p.AutoConnect(ctx); autoErr == nil {
			wait := p.AutoWait
			if wait <= 0 {
				wait = defaultAutoWait
			}
			t, werr := s.waitUp(ctx, p.Matcher, wait)
			if werr == nil {
				s.attach(t, vpn.PhaseAuto, p, hook)
				return nil
			}
			autoErr = werr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	// (3) Open the app and wait for the user.
	if err := ctx.Err(); err != nil {
		return err
	}
	_ = s.doLaunch(p.App, o.AppPath) // not fatal: the user may open it
	wait := o.WaitTimeout
	if wait <= 0 {
		wait = DefaultWaitTimeout
	}
	s.setState(vpn.AttachState{
		Phase: vpn.PhaseWaitingUser, App: p.App.DisplayName, Location: p.Location,
		Reason: ShortReason(autoErr), TimeoutSeconds: int(wait / time.Second),
	}, hook)

	t, err := s.waitUp(ctx, p.Matcher, wait)
	if err != nil {
		if errors.Is(err, errWaitTimeout) {
			return fmt.Errorf("%s (%s): %w", p.App.DisplayName, wait, vpn.ErrUserActionTimeout)
		}
		return err
	}
	s.attach(t, vpn.PhaseUser, p, hook)
	return nil
}

var errWaitTimeout = errors.New("wait timed out")

// waitUp polls until the provider's tunnel appears, ctx ends or timeout.
func (s *Session) waitUp(ctx context.Context, m Matcher, timeout time.Duration) (Tunnel, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(s.pollEvery())
	defer tick.Stop()
	for {
		if t, ok := s.detect(m); ok {
			return t, nil
		}
		select {
		case <-ctx.Done():
			return Tunnel{}, ctx.Err()
		case <-deadline.C:
			return Tunnel{}, errWaitTimeout
		case <-tick.C:
		}
	}
}

// WaitDown polls until the provider's tunnel is gone, ctx ends or timeout.
func (s *Session) WaitDown(ctx context.Context, m Matcher, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(s.pollEvery())
	defer tick.Stop()
	for {
		if _, ok := s.detect(m); !ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errWaitTimeout
		case <-tick.C:
		}
	}
}

// Disconnect cancels a pending connect wait, then disconnects through
// p.Disconnect when available. Without one (or when it fails / the
// interface stays up) it opens the app, sets PhaseWaitingDisconnect and
// returns an error wrapping vpn.ErrManualDisconnect; Status() keeps
// reporting connected until the interface drops.
func (s *Session) Disconnect(ctx context.Context, p Provider, o Options) error {
	s.mu.Lock()
	if s.cancelWait != nil {
		s.cancelWait()
	}
	s.mu.Unlock()

	if _, ok := s.detect(p.Matcher); !ok {
		s.markDown(nil)
		return nil
	}

	var cause error
	if p.Disconnect != nil {
		if cause = p.Disconnect(ctx); cause == nil {
			wait := p.DisconnectWait
			if wait <= 0 {
				wait = defaultDisconnectWait
			}
			if cause = s.WaitDown(ctx, p.Matcher, wait); cause == nil {
				s.markDown(nil)
				return nil
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	_ = s.doLaunch(p.App, o.AppPath)
	s.setState(vpn.AttachState{
		Phase: vpn.PhaseWaitingDisconnect, App: p.App.DisplayName, Reason: ShortReason(cause),
	}, nil)
	return fmt.Errorf("%s: %w", p.App.DisplayName, vpn.ErrManualDisconnect)
}

// markDown records a dropped tunnel with the given state (nil clears it).
func (s *Session) markDown(st *vpn.AttachState) {
	s.mu.Lock()
	s.status = vpn.StatusDisconnected
	s.tunnel = Tunnel{}
	s.connectedAt = time.Time{}
	s.state = st
	s.mu.Unlock()
}

// Status reports the tunnel state by interface detection: connecting while
// Connect runs; otherwise connected iff the provider's interface is up (a
// tunnel the user brought up in the app is attached, PhaseAttached). A
// tunnel that drops without a pending `down` becomes PhaseDropped when
// manualReconnect (only the user can bring it back), else clears.
func (s *Session) Status(p Provider, manualReconnect bool) vpn.Status {
	s.mu.Lock()
	connecting := s.status == vpn.StatusConnecting
	s.mu.Unlock()
	if connecting {
		return vpn.StatusConnecting
	}
	t, ok := s.detect(p.Matcher)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == vpn.StatusConnecting { // Connect started meanwhile
		return vpn.StatusConnecting
	}
	if !ok {
		if s.status == vpn.StatusConnected {
			s.status = vpn.StatusDisconnected
			s.tunnel = Tunnel{}
			s.connectedAt = time.Time{}
			wasDown := s.state != nil && s.state.Phase == vpn.PhaseWaitingDisconnect
			s.state = nil
			if manualReconnect && !wasDown {
				s.state = &vpn.AttachState{Phase: vpn.PhaseDropped, App: p.App.DisplayName, Since: time.Now()}
			}
		} else if s.state != nil && s.state.Phase == vpn.PhaseWaitingDisconnect {
			s.state = nil
		}
		return s.status.Normalize()
	}
	wasUp := s.status == vpn.StatusConnected
	s.attachLocked(t)
	if !wasUp || s.state == nil || s.state.Phase == vpn.PhaseDropped {
		s.state = &vpn.AttachState{Phase: vpn.PhaseAttached, App: p.App.DisplayName, Since: time.Now()}
	}
	return s.status.Normalize()
}

// Info returns the attached tunnel's info, or nil when not connected.
// Call Status first to refresh.
func (s *Session) Info() *vpn.TunnelInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status != vpn.StatusConnected {
		return nil
	}
	return &vpn.TunnelInfo{
		InterfaceName: s.tunnel.Name,
		AssignedIP:    s.tunnel.IP,
		ConnectedAt:   s.connectedAt,
	}
}

// AttachState returns a copy of the current attach state, or nil.
func (s *Session) AttachState() *vpn.AttachState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return nil
	}
	st := *s.state
	return &st
}
