// Package monitor aggregates tunnel metrics and dispatches health alerts.
package monitor

import (
	"sync"
	"time"

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
}

// Collector aggregates metrics from all registered VPN adapters.
type Collector struct {
	mu       sync.RWMutex
	adapters map[string]vpn.VPNAdapter
	metrics  map[string]*TunnelMetrics
	stop     chan struct{}
}

// NewCollector creates a metrics collector for the given adapters.
func NewCollector(adapters map[string]vpn.VPNAdapter) *Collector {
	return &Collector{
		adapters: adapters,
		metrics:  make(map[string]*TunnelMetrics),
		stop:     make(chan struct{}),
	}
}

// Start begins collecting metrics every interval.
func (c *Collector) Start(interval time.Duration) {
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

// Snapshot returns a copy of the current metrics for all tunnels.
func (c *Collector) Snapshot() map[string]TunnelMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[string]TunnelMetrics, len(c.metrics))
	for k, v := range c.metrics {
		out[k] = *v
	}
	return out
}

func (c *Collector) collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for name, adapter := range c.adapters {
		m := &TunnelMetrics{
			Name:      name,
			Status:    adapter.Status(),
			UpdatedAt: now,
		}

		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			m.InterfaceName = info.InterfaceName
			m.BytesSent = info.BytesSent
			m.BytesReceived = info.BytesReceived
			m.ConnectedAt = info.ConnectedAt
			if !info.ConnectedAt.IsZero() {
				m.UptimeSeconds = now.Sub(info.ConnectedAt).Seconds()
			}
			if info.AssignedIP != nil {
				m.AssignedIP = info.AssignedIP.String()
			}
		}

		c.metrics[name] = m
	}
}
