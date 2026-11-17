package security

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultPublicIPEndpoint answers GET with the caller's public IP as plain
// text (https://www.ipify.org).
const DefaultPublicIPEndpoint = "https://api.ipify.org"

const (
	// DefaultPublicIPMinInterval is the minimum gap between two automatic
	// lookups (tunnel connect/disconnect events). Manual checks skip it.
	DefaultPublicIPMinInterval = 30 * time.Second
	// defaultPublicIPSettle waits for routes to settle after a tunnel
	// event before looking up the new egress IP.
	defaultPublicIPSettle = 3 * time.Second
	publicIPTimeout       = 5 * time.Second
	// publicIPMaxBody: an IPv6 address plus whitespace fits easily.
	publicIPMaxBody = 64
)

// ErrPublicIPDisabled is returned (as LeakResult.Disabled) when the user has
// not opted in to the external public IP lookup.
var ErrPublicIPDisabled = errors.New("public IP check is disabled")

// LeakResult holds the result of a single public IP / leak check.
type LeakResult struct {
	PublicIP  string
	HasLeak   bool
	Reason    string
	CheckedAt time.Time
	// Disabled: the lookup is off; no request was made.
	Disabled bool
}

// LeakTesterOptions configures a LeakTester.
type LeakTesterOptions struct {
	Enabled     bool
	Endpoints   []string          // https URLs, tried in order; empty = default
	ExpectedIPs map[string]string // vpnName → expected public IP (leak check)
	Client      *http.Client      // nil = 5s-timeout client (tests inject a fake)
	MinInterval time.Duration     // 0 = DefaultPublicIPMinInterval
	SettleDelay time.Duration     // 0 = 3s; negative = no delay
}

// LeakTester looks up the public IP through an external service, only when
// the user opted in and only on demand: CheckNow (user action) or Trigger
// (tunnel connect/disconnect, debounced). It never polls. The kill switch
// and DNS guard do not depend on it.
type LeakTester struct {
	mu          sync.Mutex
	enabled     bool
	endpoints   []string
	expectedIPs map[string]string
	client      *http.Client
	minInterval time.Duration
	settle      time.Duration
	onResult    func(LeakResult)

	last    *LeakResult
	lastRun time.Time
	timer   *time.Timer
	stopped bool
	checkMu sync.Mutex // one lookup at a time
}

// NewLeakTester creates a tester. A disabled tester never makes a request.
func NewLeakTester(opts LeakTesterOptions) *LeakTester {
	t := &LeakTester{
		client:      opts.Client,
		minInterval: opts.MinInterval,
		settle:      opts.SettleDelay,
	}
	if t.client == nil {
		t.client = &http.Client{Timeout: publicIPTimeout}
	}
	if t.minInterval <= 0 {
		t.minInterval = DefaultPublicIPMinInterval
	}
	if t.settle == 0 {
		t.settle = defaultPublicIPSettle
	} else if t.settle < 0 {
		t.settle = 0
	}
	t.Configure(opts.Enabled, opts.Endpoints, opts.ExpectedIPs)
	return t
}

// Configure applies a config change at runtime. Disabling cancels a pending
// automatic lookup and forgets the last result (the IP is no longer shown).
func (t *LeakTester) Configure(enabled bool, endpoints []string, expectedIPs map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = enabled
	t.endpoints = nil
	for _, ep := range endpoints {
		if ep = strings.TrimSpace(ep); ep != "" {
			t.endpoints = append(t.endpoints, ep)
		}
	}
	if len(t.endpoints) == 0 {
		t.endpoints = []string{DefaultPublicIPEndpoint}
	}
	t.expectedIPs = expectedIPs
	if !enabled {
		t.last = nil
		if t.timer != nil {
			t.timer.Stop()
			t.timer = nil
		}
	}
}

// SetOnResult registers a callback run after every completed lookup.
func (t *LeakTester) SetOnResult(fn func(LeakResult)) {
	t.mu.Lock()
	t.onResult = fn
	t.mu.Unlock()
}

// Enabled reports whether the user opted in to the lookup.
func (t *LeakTester) Enabled() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.enabled
}

// Host returns the host of the first endpoint (shown in the opt-in notice).
func (t *LeakTester) Host() string {
	if t == nil {
		return hostOf(DefaultPublicIPEndpoint)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return hostOf(t.endpoints[0])
}

// Trigger schedules an automatic lookup after a tunnel event. Calls are
// coalesced: at most one lookup per MinInterval, and a trigger inside that
// window runs once when it ends, so the shown IP is never stale.
func (t *LeakTester) Trigger() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.enabled || t.stopped || t.timer != nil {
		return
	}
	wait := t.settle
	if !t.lastRun.IsZero() {
		if gap := t.minInterval - time.Since(t.lastRun); gap > wait {
			wait = gap
		}
	}
	t.timer = time.AfterFunc(wait, func() {
		t.mu.Lock()
		t.timer = nil
		run := t.enabled && !t.stopped
		t.mu.Unlock()
		if run {
			t.CheckNow()
		}
	})
}

// Stop cancels any pending automatic lookup; later Triggers are ignored.
func (t *LeakTester) Stop() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
}

// CheckNow performs a lookup now (user action) and stores the result. When
// the lookup is disabled it returns Disabled without any request.
func (t *LeakTester) CheckNow() LeakResult {
	if t == nil || !t.Enabled() {
		return LeakResult{Disabled: true, CheckedAt: time.Now(), Reason: ErrPublicIPDisabled.Error()}
	}
	t.checkMu.Lock()
	defer t.checkMu.Unlock()

	t.mu.Lock()
	endpoints := append([]string(nil), t.endpoints...)
	expected := t.expectedIPs
	t.lastRun = time.Now()
	t.mu.Unlock()

	result := LeakResult{CheckedAt: time.Now()}
	ip, err := t.fetchPublicIP(endpoints)
	if err != nil {
		result.Reason = fmt.Sprintf("IP check failed: %v", err) // a network error is not a leak
	} else {
		result.PublicIP = ip
		for vpnName, expectedIP := range expected {
			if expectedIP != "" && !strings.EqualFold(ip, expectedIP) {
				result.HasLeak = true
				result.Reason = fmt.Sprintf("expected IP %s for VPN %q, got %s", expectedIP, vpnName, ip)
				break
			}
		}
	}

	t.mu.Lock()
	if !t.enabled { // turned off while the request was in flight
		t.mu.Unlock()
		return LeakResult{Disabled: true, CheckedAt: result.CheckedAt, Reason: ErrPublicIPDisabled.Error()}
	}
	stored := result
	t.last = &stored
	cb := t.onResult
	t.mu.Unlock()
	if cb != nil {
		cb(result)
	}
	return result
}

// LastResult returns the most recent result (nil before the first lookup or
// while disabled).
func (t *LeakTester) LastResult() *LeakResult {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.enabled || t.last == nil {
		return nil
	}
	r := *t.last
	return &r
}

func (t *LeakTester) fetchPublicIP(endpoints []string) (string, error) {
	var errs []error
	for _, ep := range endpoints {
		ip, err := t.fetchOne(ep)
		if err == nil {
			return ip, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", hostOf(ep), err))
	}
	return "", errors.Join(errs...)
}

func (t *LeakTester) fetchOne(endpoint string) (string, error) {
	if err := checkEndpoint(endpoint); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), publicIPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/plain")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, publicIPMaxBody+1))
	if err != nil {
		return "", err
	}
	if len(body) > publicIPMaxBody {
		return "", errors.New("response too large")
	}
	addr, err := netip.ParseAddr(strings.TrimSpace(string(body)))
	if err != nil {
		return "", errors.New("response is not an IP address")
	}
	return addr.Unmap().String(), nil
}

// checkEndpoint re-checks at request time what config validation enforces:
// https only, with a host and no credentials.
func checkEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Hostname() == "" || u.User != nil {
		return errors.New("endpoint must be an https:// URL")
	}
	return nil
}

func hostOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return raw
}
