package monitor

import (
	"context"
	"net"
	"net/netip"
	"sort"
	"sync"
	"time"
)

// Recent destinations (ADR-0054): which hosts this computer connected to
// lately, through which tunnel, by which program. Privacy rules:
//   - kept only in this struct's memory: never written to disk or logs,
//     never sent off the machine; served only on the authenticated local
//     API (GET /api/v1/connections/recent);
//   - bounded (RecentMax entries) and forgotten RecentTTL after it was
//     last seen; turning the setting off empties it at once.
const (
	RecentMax   = 500
	RecentTTL   = time.Hour
	RecentEvery = 15 * time.Second
)

// RecentConnection is one destination in the list.
type RecentConnection struct {
	// Host is the name the address was resolved from for a policy domain,
	// "" when unknown (clients then show RemoteIP).
	Host     string `json:"host,omitempty"`
	RemoteIP string `json:"remote_ip"`
	Port     uint16 `json:"port"`
	// Process is the program's executable name, "" when the OS does not
	// tell a standard user.
	Process string `json:"process,omitempty"`
	// Via is the Kongtrol profile whose tunnel the connection left
	// through, "" for direct (not through a Kongtrol tunnel).
	Via string `json:"via,omitempty"`
	// Interface is the local network interface it left through.
	Interface string    `json:"interface,omitempty"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	// Count is how many connections were seen opening to it (a
	// connection open across several checks counts once).
	Count int `json:"count"`
}

type recentKey struct {
	remote  netip.AddrPort
	process string
	via     string
	iface   string
}

type recentEntry struct {
	RecentConnection
	ports map[uint16]bool // local ports open at the last check
}

// RecentConnections keeps the list. The zero value is not usable; use
// NewRecentConnections.
type RecentConnections struct {
	// Host names an address (policy resolver), "" when unknown.
	Host func(netip.Addr) string
	// Via names the Kongtrol profile whose tunnel owns local address a
	// ("" = not a tunnel).
	Via func(a netip.Addr) string

	mu      sync.Mutex
	enabled bool
	entries map[recentKey]*recentEntry
	max     int
	ttl     time.Duration

	supported bool
	now       func() time.Time
	conns     func() ([]TCPConn, error)
	names     func([]TCPConn) map[TCPConn]string
	ifaces    func() map[netip.Addr]string
}

// NewRecentConnections returns an empty list, on or off.
func NewRecentConnections(enabled bool) *RecentConnections {
	return &RecentConnections{
		enabled:   enabled,
		entries:   map[recentKey]*recentEntry{},
		max:       RecentMax,
		ttl:       RecentTTL,
		supported: tcpConnectionsSupported,
		now:       time.Now,
		conns:     tcpConnections,
		names:     processNames,
		ifaces:    localInterfaceNames,
	}
}

// Supported reports whether this OS can list connections without admin.
func (r *RecentConnections) Supported() bool { return r.supported }

// Enabled reports whether the list is on.
func (r *RecentConnections) Enabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enabled
}

// SetEnabled turns the list on or off; off forgets everything at once.
func (r *RecentConnections) SetEnabled(on bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = on
	if !on {
		clear(r.entries)
	}
}

// Run checks every interval until ctx ends (nothing is read while off).
func (r *RecentConnections) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			r.SetEnabled(false) // forget on shutdown too
			return
		case <-t.C:
			r.Check()
		}
	}
}

// Check reads the open connections once and updates the list.
func (r *RecentConnections) Check() {
	if !r.Enabled() || !r.supported {
		return
	}
	conns, err := r.conns()
	if err != nil {
		return
	}
	var keep []TCPConn
	for _, c := range conns {
		a := c.Remote.Addr()
		if !a.IsValid() || a.IsLoopback() || a.IsUnspecified() {
			continue
		}
		keep = append(keep, c)
	}
	names := r.names(keep)
	ifaces := r.ifaces()
	now := r.now()

	type seen struct{ ports map[uint16]bool }
	cur := map[recentKey]*seen{}
	for _, c := range keep {
		local := c.Local.Addr()
		k := recentKey{remote: c.Remote, process: names[c], iface: ifaces[local]}
		if r.Via != nil {
			k.via = r.Via(local)
		}
		s := cur[k]
		if s == nil {
			s = &seen{ports: map[uint16]bool{}}
			cur[k] = s
		}
		s.ports[c.Local.Port()] = true
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.enabled {
		return // turned off while reading
	}
	for k, s := range cur {
		e := r.entries[k]
		if e == nil {
			r.evictLocked()
			host := ""
			if r.Host != nil {
				host = r.Host(k.remote.Addr())
			}
			e = &recentEntry{RecentConnection: RecentConnection{
				Host: host, RemoteIP: k.remote.Addr().String(), Port: k.remote.Port(),
				Process: k.process, Via: k.via, Interface: k.iface, FirstSeen: now,
			}}
			r.entries[k] = e
		} else if e.Host == "" && r.Host != nil {
			e.Host = r.Host(k.remote.Addr())
		}
		for p := range s.ports {
			if !e.ports[p] {
				e.Count++
			}
		}
		e.ports = s.ports
		e.LastSeen = now
	}
	// Connections that closed: a later one on the same local port is new.
	for k, e := range r.entries {
		if cur[k] == nil {
			e.ports = nil
		}
	}
	r.expireLocked(now)
}

// evictLocked makes room for one entry: the one seen longest ago goes.
func (r *RecentConnections) evictLocked() {
	for len(r.entries) >= r.max {
		var oldest recentKey
		var at time.Time
		first := true
		for k, e := range r.entries {
			if first || e.LastSeen.Before(at) {
				oldest, at, first = k, e.LastSeen, false
			}
		}
		delete(r.entries, oldest)
	}
}

func (r *RecentConnections) expireLocked(now time.Time) {
	for k, e := range r.entries {
		if now.Sub(e.LastSeen) > r.ttl {
			delete(r.entries, k)
		}
	}
}

// List returns up to limit destinations (0 = all), most recent first.
func (r *RecentConnections) List(limit int) []RecentConnection {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expireLocked(r.now())
	out := make([]RecentConnection, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.RecentConnection)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastSeen.Equal(out[j].LastSeen) {
			return out[i].LastSeen.After(out[j].LastSeen)
		}
		if out[i].RemoteIP != out[j].RemoteIP {
			return out[i].RemoteIP < out[j].RemoteIP
		}
		return out[i].Port < out[j].Port
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// localInterfaceNames maps this computer's addresses to interface names.
func localInterfaceNames() map[netip.Addr]string {
	out := map[netip.Addr]string{}
	ifs, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, ifc := range ifs {
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if na, ok := netip.AddrFromSlice(ip); ok {
				out[na.Unmap()] = ifc.Name
			}
		}
	}
	return out
}
