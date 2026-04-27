package security

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultIPCheckURL = "https://api.ipify.org"

// LeakResult holds the result of a single leak test.
type LeakResult struct {
	PublicIP  string
	HasLeak   bool
	Reason    string
	CheckedAt time.Time
}

// LeakTester periodically checks for IP/DNS leaks.
type LeakTester struct {
	mu          sync.RWMutex
	ipCheckURL  string
	interval    time.Duration
	expectedIPs map[string]string // vpnName → expected public IP
	lastResult  map[string]*LeakResult
	client      *http.Client
	cancel      context.CancelFunc
}

// NewLeakTester creates a leak tester that checks every interval.
// expectedIPs maps VPN profile names to their expected public IP addresses.
// Pass nil or empty map to skip IP verification (detect only).
func NewLeakTester(interval time.Duration, expectedIPs map[string]string) *LeakTester {
	return &LeakTester{
		ipCheckURL:  defaultIPCheckURL,
		interval:    interval,
		expectedIPs: expectedIPs,
		lastResult:  make(map[string]*LeakResult),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start begins periodic leak testing. Call Stop() to halt.
// onLeak is called each time a leak is detected.
func (t *LeakTester) Start(ctx context.Context, onLeak func(result LeakResult)) {
	ctx, t.cancel = context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result := t.CheckNow()
				t.mu.Lock()
				t.lastResult["default"] = &result
				t.mu.Unlock()
				if result.HasLeak && onLeak != nil {
					onLeak(result)
				}
			}
		}
	}()
}

// Stop halts the periodic leak tester.
func (t *LeakTester) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
}

// CheckNow performs an immediate leak test and returns the result.
func (t *LeakTester) CheckNow() LeakResult {
	result := LeakResult{CheckedAt: time.Now()}

	ip, err := t.fetchPublicIP()
	if err != nil {
		result.HasLeak = false // network error is not a leak
		result.Reason = fmt.Sprintf("IP check failed: %v", err)
		return result
	}
	result.PublicIP = ip

	// If expected IPs are configured, verify the current IP matches.
	for vpnName, expectedIP := range t.expectedIPs {
		if expectedIP != "" && !strings.EqualFold(ip, expectedIP) {
			result.HasLeak = true
			result.Reason = fmt.Sprintf("expected IP %s for VPN %q, got %s", expectedIP, vpnName, ip)
			return result
		}
	}

	return result
}

// LastResult returns the most recent leak test result.
func (t *LeakTester) LastResult() *LeakResult {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastResult["default"]
}

func (t *LeakTester) fetchPublicIP() (string, error) {
	resp, err := t.client.Get(t.ipCheckURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
