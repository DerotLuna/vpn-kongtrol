package privops

import (
	"bytes"
	"context"
	"net/netip"
	"path/filepath"
	"strings"
)

func CheckDefaultEgress(profileType, path, tunnelName string) DefaultEgressCheck {
	return checkWireGuardDefaultEgress(profileType, path, tunnelName)
}

// WireGuard for Windows installs these routes and removes them with the tunnel.
// Its /0 firewall would block even the specific routes set for other tunnels.
// Use WireGuard's documented pair of /1s instead, on the private copy only.
func prepareWireGuardDefaultEgress(data []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(data) + 40)
	for line := range bytes.SplitSeq(data, []byte{'\n'}) {
		setting, _, _ := bytes.Cut(line, []byte{'#'})
		key, value, found := bytes.Cut(setting, []byte{'='})
		key = bytes.TrimSpace(key)
		// The compatibility check accepts SaveConfig=false from wg-quick.
		// The Windows parser has no SaveConfig setting.
		if found && bytes.EqualFold(key, []byte("SaveConfig")) {
			continue
		}
		if !found || !bytes.EqualFold(key, []byte("AllowedIPs")) {
			out.Write(line)
			out.WriteByte('\n')
			continue
		}
		out.WriteString("AllowedIPs = ")
		var routes []string
		for item := range bytes.SplitSeq(value, []byte{','}) {
			cidr := string(bytes.TrimSpace(item))
			prefix, _ := netip.ParsePrefix(cidr)
			switch prefix {
			case netip.PrefixFrom(netip.IPv4Unspecified(), 0):
				routes = append(routes, "0.0.0.0/1", "128.0.0.0/1")
			case netip.PrefixFrom(netip.IPv6Unspecified(), 0):
				routes = append(routes, "::/1", "8000::/1")
			default:
				routes = append(routes, cidr)
			}
		}
		out.WriteString(strings.Join(routes, ", "))
		out.WriteByte('\n')
	}
	return out.Bytes()
}

func checkNativeDefaultRules(ctx context.Context, replacing string) error {
	// Exclude the host's native /0 gateways. Routes with a /1 prefix
	// can compete with these tunnels even when they belong to a physical NIC.
	const script = `$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [Text.Encoding]::UTF8
$hardware = @(Get-NetAdapter -IncludeHidden | Where-Object HardwareInterface | ForEach-Object ifIndex)
Get-NetRoute -PolicyStore ActiveStore | Where-Object {
  $_.DestinationPrefix -match '/1$' -or
  ($_.DestinationPrefix -match '/0$' -and $_.InterfaceIndex -notin $hardware -and
   $_.NextHop -in @('0.0.0.0','::'))
} | ForEach-Object { '{0}|{1}|{2}' -f $_.InterfaceIndex, $_.DestinationPrefix, $_.InterfaceAlias }`
	ps := filepath.Join(systemDir(), "WindowsPowerShell", "v1.0", "powershell.exe")
	output, err := run(ctx, ps, "-NoProfile", "-NonInteractive", "-Command", script)
	if err != nil {
		return err
	}
	if conflictingWindowsDefaultRoutes(output, replacing) {
		return ErrWGDefaultEgressConflict
	}
	return nil
}

func conflictingWindowsDefaultRoutes(output, replacing string) bool {
	for line := range strings.SplitSeq(output, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 3)
		if len(parts) != 3 || (replacing != "" && strings.EqualFold(parts[2], replacing)) {
			continue
		}
		if p, err := netip.ParsePrefix(parts[1]); err == nil && p.Bits() <= 1 {
			return true
		}
	}
	return false
}
