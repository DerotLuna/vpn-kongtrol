package privops

import (
	"path"
	"strings"
)

// systemd-resolved (Ubuntu, Fedora, Debian with systemd-resolved): when
// /etc/resolv.conf is a link into /run/systemd/resolve/, resolved owns DNS
// and the file is only its stub. The DNS guard then sets DNS per link with
// resolvectl instead of rewriting the file (ADR-0058):
//
//	resolvectl dns <tunnel> <servers…>
//	resolvectl domain <tunnel> ~.        all names to the tunnel's servers
//	resolvectl default-route <tunnel> yes
//
// and undoes it with `resolvectl revert <tunnel>`. resolved drops a link's
// settings by itself when the interface disappears, so a tunnel that
// crashes never leaves the machine with a dead resolver (resolvectl(1)).
// This is the same mapping resolved's own resolvconf -x uses (wg-quick).

// resolvedStubDir is where systemd-resolved keeps its resolv.conf files.
const resolvedStubDir = "/run/systemd/resolve/"

// resolvConfManagedByResolved reports whether the resolved target of
// /etc/resolv.conf is one of systemd-resolved's files.
func resolvConfManagedByResolved(target string) bool {
	return strings.HasPrefix(path.Clean(target), resolvedStubDir)
}

// resolvectlSetArgs are the resolvectl calls that point every lookup at
// the tunnel's servers. The last one (default-route) is best effort: older
// systemd (< 240) has no such verb.
func resolvectlSetArgs(req DNSRequest) [][]string {
	dns := []string{"dns", req.Interface}
	for _, s := range req.Servers {
		dns = append(dns, s.String())
	}
	return [][]string{
		dns,
		{"domain", req.Interface, "~."},
		{"default-route", req.Interface, "yes"},
	}
}
