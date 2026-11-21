package monitor

import (
	"context"
	"net"
	"net/netip"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// SplitDNSManager injects policy-resolved domains into system hosts file
// so regular applications resolve them without calling the API.
type SplitDNSManager struct {
	cfg      *config.Config
	resolver *PolicyResolver
	interval time.Duration
	log      *zap.Logger
	cancel   context.CancelFunc
}

func NewSplitDNSManager(cfg *config.Config, resolver *PolicyResolver, interval time.Duration, log *zap.Logger) *SplitDNSManager {
	return &SplitDNSManager{
		cfg:      cfg,
		resolver: resolver,
		interval: interval,
		log:      log,
	}
}

func (m *SplitDNSManager) Start(ctx context.Context) {
	if m == nil || m.resolver == nil || m.interval <= 0 {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	go m.loop(ctx)
}

func (m *SplitDNSManager) Stop() {
	if m == nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	_ = m.restore()
}

func (m *SplitDNSManager) loop(ctx context.Context) {
	m.sync()
	t := time.NewTicker(m.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.sync()
		}
	}
}

func (m *SplitDNSManager) sync() {
	hostMap := map[string]string{}

	for _, pol := range m.cfg.Policies {
		for _, domainPattern := range pol.Match.Domains {
			for _, domain := range expandDomainPattern(domainPattern) {
				ips, err := m.resolver.ResolveDomainViaProfile(pol.Via, domain)
				if err != nil || len(ips) == 0 {
					continue
				}
				if ip := pickPreferredIP(ips); ip != nil {
					hostMap[strings.TrimSuffix(domain, ".")] = ip.String()
				}
			}
		}
	}

	if err := m.writeHosts(hostMap); err != nil {
		m.log.Warn("splitdns: cannot update hosts", zap.Error(err))
	}
}

func (m *SplitDNSManager) restore() error {
	return m.writeHosts(map[string]string{})
}

func pickPreferredIP(ips []net.IP) net.IP {
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}
	if len(ips) > 0 {
		return ips[0]
	}
	return nil
}

// hostEntries converts domain → IP into the hosts block entries, sorted by
// domain; invalid IPs are skipped.
func hostEntries(entries map[string]string) []privops.HostEntry {
	domains := make([]string, 0, len(entries))
	for d := range entries {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	out := make([]privops.HostEntry, 0, len(domains))
	for _, d := range domains {
		ip, err := netip.ParseAddr(entries[d])
		if err != nil {
			continue
		}
		out = append(out, privops.HostEntry{IP: ip.Unmap(), Host: strings.ToLower(d)})
	}
	return out
}

// writeHosts replaces only the Kongtrol-marked block of the system hosts
// file through privops (helper or in-process when elevated). Empty entries
// remove the block.
func (m *SplitDNSManager) writeHosts(entries map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ops, err := privops.Default(ctx)
	if err != nil {
		return err
	}
	return ops.WriteHostsBlock(ctx, hostEntries(entries))
}
