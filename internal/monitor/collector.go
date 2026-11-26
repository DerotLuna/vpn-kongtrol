// Package monitor aggregates tunnel metrics and dispatches health alerts.
package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/netstats"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// TunnelMetrics holds a point-in-time snapshot of a tunnel's statistics.
type TunnelMetrics struct {
	Name          string
	Status        vpn.Status
	InterfaceName string
	AssignedIP    string
	ConnectedAt   time.Time
	UptimeSeconds float64
	BytesSent     uint64
	BytesReceived uint64
	LatencyMS     int64
	UpdatedAt     time.Time
	// LastError is the message of the most recent failed asynchronous
	// connect (API, dashboard, tray, `up`) and LastErrorAt when it failed.
	// Both are cleared when a new connect starts or one succeeds.
	LastError   string    `json:"last_error,omitempty"`
	LastErrorAt time.Time `json:"last_error_at,omitzero"`
	// LastErrorKey is the i18n key of a one-line "what to do" message for
	// LastError when one exists (e.g. "service.error.not_installed" when
	// the privileged helper is missing). Clients render it translated
	// instead of the raw English LastError.
	LastErrorKey string `json:"last_error_key,omitempty"`
	// Attach is a provider adapter's (attach mode) phase: how the tunnel
	// came up (attached / auto / user) or what the user must do
	// (waiting_user / waiting_disconnect / dropped). Clients render it with
	// Attach.MessageKey() + i18n in their own language.
	Attach *vpn.AttachState `json:"attach,omitempty"`
	// Health says whether a connected tunnel's server answers (WireGuard:
	// latest handshake, ADR-0030): ok / idle / no_response / unknown.
	// Clients render Health.MessageKey() (+ ReasonKey) with i18n.
	Health    *vpn.TunnelHealth `json:"health,omitempty"`
	Restoring bool              `json:"restoring,omitempty"`
}

// interfaceCounters is the OS byte-counter source; a variable so tests can
// stub it.
var interfaceCounters = netstats.Counters

// connectFailure is the last failed connect recorded for one profile.
type connectFailure struct {
	msg string
	key string
	at  time.Time
}

// ProfileHistory holds rolling historical stability metrics per profile.
type ProfileHistory struct {
	Name             string    `json:"name"`
	Samples          int64     `json:"samples"`
	ConnectedSamples int64     `json:"connected_samples"`
	UptimeSeconds    float64   `json:"uptime_seconds"`
	LastConnectedAt  time.Time `json:"last_connected_at,omitempty"`
	LastDownAt       time.Time `json:"last_down_at,omitempty"`
	Reconnects       int64     `json:"reconnects"`
	Leaks            int64     `json:"leaks"`
	LastLatencyMS    int64     `json:"last_latency_ms"`
	JitterMS         int64     `json:"jitter_ms"`
	LastReachable    bool      `json:"last_reachable"`
	LastDNSOK        bool      `json:"last_dns_ok"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Collector aggregates metrics from all registered VPN adapters.
type Collector struct {
	mu       sync.RWMutex
	adapters *vpn.Set
	metrics  map[string]*TunnelMetrics
	history  map[string]*ProfileHistory
	lastErr  map[string]connectFailure
	startup  map[string]bool
	stop     chan struct{}

	subMu sync.Mutex
	subs  map[chan struct{}]struct{}
}

// SetStartupPending reports a planned startup connect to clients without
// changing the adapter state used by routing and the watchdog.
func (c *Collector) SetStartupPending(name string, pending, reconnect bool) {
	c.mu.Lock()
	if pending {
		if c.startup == nil {
			c.startup = make(map[string]bool)
		}
		c.startup[name] = reconnect
	} else {
		delete(c.startup, name)
	}
	c.mu.Unlock()
	c.notifyChange()
}

// ClientSnapshot adds planned startup work to the real adapter snapshot.
func (c *Collector) ClientSnapshot() map[string]TunnelMetrics {
	out := c.Snapshot()
	c.mu.RLock()
	defer c.mu.RUnlock()
	for name, reconnect := range c.startup {
		if m, ok := out[name]; ok && m.Status.Normalize() != vpn.StatusConnected {
			m.Status = vpn.StatusConnecting
			m.Restoring = reconnect
			m.LastError, m.LastErrorKey = "", ""
			out[name] = m
		}
	}
	return out
}

// NewCollector creates a metrics collector for the given adapters.
func NewCollector(adapters *vpn.Set) *Collector {
	return &Collector{
		adapters: adapters,
		metrics:  make(map[string]*TunnelMetrics),
		history:  make(map[string]*ProfileHistory),
		stop:     make(chan struct{}),
	}
}

// Start begins collecting metrics every interval.
func (c *Collector) Start(interval time.Duration) {
	// Prime metrics immediately so API/CLI consumers don't see an empty first snapshot.
	c.collect()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-c.stop:
				return
			case <-ticker.C:
				c.collect()
			}
		}
	}()
}

// Stop halts metric collection.
func (c *Collector) Stop() {
	close(c.stop)
}

// Subscribe returns a channel that receives a notification each time a
// collection pass observes a meaningful change in tunnel state (status,
// interface, assigned IP, or connection time) — not on every poll tick.
// The channel is buffered size 1 and sends are non-blocking, so a slow or
// idle subscriber can't stall collection; multiple changes between reads
// simply coalesce into one pending signal. Call the returned cancel func
// to unsubscribe (e.g. when the consumer stops watching).
func (c *Collector) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	c.subMu.Lock()
	if c.subs == nil {
		c.subs = make(map[chan struct{}]struct{})
	}
	c.subs[ch] = struct{}{}
	c.subMu.Unlock()

	cancel := func() {
		c.subMu.Lock()
		delete(c.subs, ch)
		c.subMu.Unlock()
	}
	return ch, cancel
}

func (c *Collector) notifyChange() {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	for ch := range c.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Snapshot returns a copy of the current metrics for all tunnels.
func (c *Collector) Snapshot() map[string]TunnelMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[string]TunnelMetrics, len(c.metrics))
	for k, v := range c.metrics {
		m := *v
		if f, ok := c.lastErr[k]; ok {
			m.LastError = f.msg
			m.LastErrorAt = f.at
			m.LastErrorKey = f.key
		}
		out[k] = m
	}
	return out
}

// RecordConnectError stores err as profile name's last connect failure and
// notifies subscribers, so live views show it without waiting for a poll.
// A nil err clears it. Safe on a nil Collector.
func (c *Collector) RecordConnectError(name string, err error) {
	if c == nil {
		return
	}
	c.mu.Lock()
	_, had := c.lastErr[name]
	if err == nil {
		delete(c.lastErr, name)
	} else {
		if c.lastErr == nil {
			c.lastErr = make(map[string]connectFailure)
		}
		key, _ := privops.GuidanceKey(err)
		c.lastErr[name] = connectFailure{msg: err.Error(), key: key, at: time.Now()}
	}
	c.mu.Unlock()
	if err != nil || had {
		c.notifyChange()
	}
}

// ClearConnectError forgets profile name's last connect failure. Safe on a
// nil Collector.
func (c *Collector) ClearConnectError(name string) {
	c.RecordConnectError(name, nil)
}

// HistorySnapshot returns a copy of historical profile metrics.
func (c *Collector) HistorySnapshot() map[string]ProfileHistory {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[string]ProfileHistory, len(c.history))
	for k, v := range c.history {
		out[k] = *v
	}
	return out
}

// RecordReconnect increments reconnect counters for a profile.
func (c *Collector) RecordReconnect(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := c.ensureHistory(name)
	h.Reconnects++
	h.UpdatedAt = time.Now()
}

// RecordLeak increments leak counters for a profile.
func (c *Collector) RecordLeak(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := c.ensureHistory(name)
	h.Leaks++
	h.UpdatedAt = time.Now()
}

// RecordHealth updates health-derived metrics (latency, jitter, reachability, DNS).
func (c *Collector) RecordHealth(name string, latency time.Duration, reachable, dnsOK bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	h := c.ensureHistory(name)
	lat := latency.Milliseconds()

	if h.LastLatencyMS > 0 && lat > 0 {
		delta := h.LastLatencyMS - lat
		if delta < 0 {
			delta = -delta
		}
		h.JitterMS = delta
	}
	h.LastLatencyMS = lat
	h.LastReachable = reachable
	h.LastDNSOK = dnsOK
	h.UpdatedAt = now

	if m, ok := c.metrics[name]; ok {
		m.LatencyMS = lat
		m.UpdatedAt = now
	}
}

func (c *Collector) collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	changed := false
	current := c.adapters.Snapshot()
	for name, adapter := range current {
		prev, hadPrev := c.metrics[name]
		prevStatus := vpn.Status("")
		if hadPrev {
			prevStatus = prev.Status.Normalize()
		}

		m := &TunnelMetrics{
			Name:      name,
			Status:    adapter.Status().Normalize(),
			UpdatedAt: now,
		}

		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			m.InterfaceName = info.InterfaceName
			m.BytesSent = info.BytesSent
			m.BytesReceived = info.BytesReceived
			// Adapters whose own tooling can't report traffic unprivileged
			// (wg show needs admin, attach-mode providers have no API) get
			// the OS interface counters, readable by any user.
			if m.BytesSent == 0 && m.BytesReceived == 0 && m.InterfaceName != "" {
				if sent, recv, ok := interfaceCounters(m.InterfaceName); ok {
					m.BytesSent, m.BytesReceived = sent, recv
				}
			}
			m.ConnectedAt = info.ConnectedAt
			if !info.ConnectedAt.IsZero() {
				m.UptimeSeconds = now.Sub(info.ConnectedAt).Seconds()
			}
			if info.AssignedIP != nil {
				m.AssignedIP = info.AssignedIP.String()
			}
		}
		if r, ok := adapter.(vpn.AttachReporter); ok {
			m.Attach = r.AttachState()
		}
		if r, ok := adapter.(vpn.HealthReporter); ok {
			m.Health = r.TunnelHealth()
		}
		// Latency comes from the watchdog health probe (RecordHealth), once
		// per health interval; keep the last reading between probes instead of
		// wiping it on every 5s collection pass.
		if m.Status == vpn.StatusConnected {
			if h, ok := c.history[name]; ok {
				m.LatencyMS = h.LastLatencyMS
			}
		}

		if !hadPrev ||
			prevStatus != m.Status ||
			prev.InterfaceName != m.InterfaceName ||
			prev.AssignedIP != m.AssignedIP ||
			!prev.ConnectedAt.Equal(m.ConnectedAt) ||
			attachKey(prev.Attach) != attachKey(m.Attach) ||
			healthKey(prev.Health) != healthKey(m.Health) {
			changed = true
		}
		// A tunnel that came up (e.g. a watchdog reconnect) no longer has
		// a relevant last connect error.
		if m.Status == vpn.StatusConnected {
			delete(c.lastErr, name)
		}

		c.metrics[name] = m

		h := c.ensureHistory(name)
		h.Samples++
		if m.Status == vpn.StatusConnected {
			h.ConnectedSamples++
			h.UptimeSeconds += 5
			if !m.ConnectedAt.IsZero() {
				h.LastConnectedAt = m.ConnectedAt
			} else if h.LastConnectedAt.IsZero() {
				h.LastConnectedAt = now
			}
		}
		if prevStatus != "" && prevStatus != m.Status {
			switch m.Status {
			case vpn.StatusConnected:
				if !m.ConnectedAt.IsZero() {
					h.LastConnectedAt = m.ConnectedAt
				} else {
					h.LastConnectedAt = now
				}
			default:
				if prevStatus == vpn.StatusConnected {
					h.LastDownAt = now
					h.LastLatencyMS = 0 // stale once the tunnel is gone
				}
			}
		}
		h.UpdatedAt = now
	}
	// A profile removed without a restart (ADR-0053) leaves the snapshot
	// and the history at once.
	for name := range c.metrics {
		if _, ok := current[name]; !ok {
			delete(c.metrics, name)
			delete(c.history, name)
			changed = true
		}
	}

	if changed {
		c.notifyChange()
	}
}

func (c *Collector) ensureHistory(name string) *ProfileHistory {
	if h, ok := c.history[name]; ok {
		return h
	}
	h := &ProfileHistory{Name: name}
	c.history[name] = h
	return h
}

// LoadHistory loads persisted historical metrics from disk.
func (c *Collector) LoadHistory(path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	decoded := map[string]ProfileHistory{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for name, h := range decoded {
		hCopy := h
		c.history[name] = &hCopy
	}
	return nil
}

// SaveHistory persists historical metrics to disk.
func (c *Collector) SaveHistory(path string) error {
	if path == "" {
		return nil
	}
	c.mu.RLock()
	out := make(map[string]ProfileHistory, len(c.history))
	for k, v := range c.history {
		out[k] = *v
	}
	c.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// attachKey identifies an attach state for change detection.
func attachKey(a *vpn.AttachState) string {
	if a == nil {
		return ""
	}
	return string(a.Phase) + "|" + a.App + "|" + a.Reason
}

// healthKey identifies the health a client renders (state + reason).
func healthKey(h *vpn.TunnelHealth) string {
	if h == nil {
		return ""
	}
	return string(h.State) + "|" + h.ReasonKey
}

// Refresh runs one collection pass now (e.g. when a provider adapter
// changes its attach phase) so subscribers see the change without waiting
// for the next poll. Safe on a nil Collector.
func (c *Collector) Refresh() {
	if c == nil {
		return
	}
	c.collect()
}
