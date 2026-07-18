package monitor

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

const splitDNSStartMarker = "# kongtrol split-dns start"
const splitDNSEndMarker = "# kongtrol split-dns end"

var splitDNSCommonSubdomains = []string{"www.", "api.", "cdn.", "app.", "docs.", "console.", "static.", "assets."}

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
			for _, domain := range expandSplitDNSDomainPattern(domainPattern) {
				ips, err := m.resolver.ResolveDomainViaProfile(pol.Via, domain)
				if err != nil || len(ips) == 0 {
					continue
				}
				if ip := pickPreferredIP(ips); ip != nil {
					hostMap[domain] = ip.String()
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

func (m *SplitDNSManager) writeHosts(entries map[string]string) error {
	path := hostsFilePath()
	if path == "" {
		return fmt.Errorf("unsupported OS for hosts file")
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	clean := removeSplitDNSBlock(string(current))
	block := renderSplitDNSBlock(entries)
	next := clean
	if strings.TrimSpace(block) != "" {
		if !strings.HasSuffix(next, "\n") {
			next += "\n"
		}
		next += block
	}
	return os.WriteFile(path, []byte(next), 0o644)
}

func renderSplitDNSBlock(entries map[string]string) string {
	if len(entries) == 0 {
		return ""
	}
	var domains []string
	for d := range entries {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	var b strings.Builder
	b.WriteString(splitDNSStartMarker + "\n")
	for _, d := range domains {
		b.WriteString(entries[d] + " " + d + "\n")
	}
	b.WriteString(splitDNSEndMarker + "\n")
	return b.String()
}

func removeSplitDNSBlock(in string) string {
	var out []string
	inBlock := false
	for _, line := range strings.Split(in, "\n") {
		trim := strings.TrimSpace(line)
		if trim == splitDNSStartMarker {
			inBlock = true
			continue
		}
		if trim == splitDNSEndMarker {
			inBlock = false
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

func expandSplitDNSDomainPattern(pattern string) []string {
	p := strings.TrimSpace(strings.ToLower(pattern))
	if p == "" {
		return nil
	}
	if strings.HasPrefix(p, "*.") {
		base := strings.TrimPrefix(p, "*.")
		targets := []string{base}
		for _, prefix := range splitDNSCommonSubdomains {
			targets = append(targets, prefix+base)
		}
		return targets
	}
	return []string{p}
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

func hostsFilePath() string {
	switch runtime.GOOS {
	case "windows":
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return root + `\System32\drivers\etc\hosts`
	case "linux", "darwin":
		return "/etc/hosts"
	default:
		return ""
	}
}
