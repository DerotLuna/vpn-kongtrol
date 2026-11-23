package privops

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"net"
	"net/netip"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrWGDefaultEgressConflict means another managed default may lose its rules.
var ErrWGDefaultEgressConflict = errors.New("a conflicting WireGuard default route is active")

// DefaultEgressCheck describes a profile file, independently of live routing.
type DefaultEgressCheck struct {
	Compatible bool   `json:"compatible"`
	Reason     string `json:"reason,omitempty"`
}

func checkWireGuardDefaultEgress(profileType, path, tunnelName string) DefaultEgressCheck {
	if profileType != "wireguard" {
		return DefaultEgressCheck{Reason: "unsupported_profile"}
	}
	if path == "" {
		return DefaultEgressCheck{Reason: "config_unavailable"}
	}
	abs, err := filepath.Abs(path)
	if err != nil || ValidateConfigPath(abs, ".conf") != nil {
		return DefaultEgressCheck{Reason: "config_unavailable"}
	}
	if tunnelName == "" {
		tunnelName = strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	}
	if ValidateWGName(tunnelName) != nil {
		return DefaultEgressCheck{Reason: "invalid_config"}
	}
	data, _, err := readTrustedFile(selfCaller(), abs, MaxConfigSize, nil)
	if err != nil {
		return DefaultEgressCheck{Reason: "config_unavailable"}
	}
	defer clear(data)
	return inspectWireGuardDefaultEgress(data)
}

func inspectWireGuardDefaultEgress(data []byte) DefaultEgressCheck {
	if ValidateWireGuardConfig(data) != nil {
		return DefaultEgressCheck{Reason: "invalid_config"}
	}

	var address4, address6, default4, default6, dns bool
	var peers int
	var inPeer bool
	seen := map[string]bool{}
	lines := bufio.NewScanner(bytes.NewReader(data))
	for lines.Scan() {
		line, _, _ := strings.Cut(lines.Text(), "#")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPeer = strings.EqualFold(line, "[Peer]")
			if inPeer {
				peers++
				if peers > 1 {
					return DefaultEgressCheck{Reason: "peer_count"}
				}
			} else if !strings.EqualFold(line, "[Interface]") || peers > 0 {
				return DefaultEgressCheck{Reason: "invalid_config"}
			}
			continue
		}

		key, value, _ := strings.Cut(line, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		// Keep this check to one canonical setting per key. Both native
		// defaults can be listed together on the peer's AllowedIPs line.
		if seen[key] {
			return DefaultEgressCheck{Reason: "unsupported_options"}
		}
		seen[key] = true
		switch key {
		case "address", "allowedips":
			for _, item := range strings.Split(value, ",") {
				prefix, err := netip.ParsePrefix(strings.TrimSpace(item))
				if err != nil || prefix.Addr().Is4In6() {
					return DefaultEgressCheck{Reason: "invalid_config"}
				}
				if key == "address" && prefix.Addr().IsGlobalUnicast() {
					address4 = address4 || prefix.Addr().Is4()
					address6 = address6 || prefix.Addr().Is6()
				}
				if key == "allowedips" && prefix == prefix.Masked() && prefix.Bits() == 0 {
					default4 = default4 || prefix.Addr().Is4()
					default6 = default6 || prefix.Addr().Is6()
				}
			}
		case "dns":
			for _, item := range strings.Split(value, ",") {
				ip, err := netip.ParseAddr(strings.TrimSpace(item))
				if err == nil && ip.IsGlobalUnicast() && !ip.Is4In6() && ip.Zone() == "" {
					dns = true
				}
			}
		case "table":
			if value != "auto" {
				return DefaultEgressCheck{Reason: "unsupported_options"}
			}
		case "fwmark":
			return DefaultEgressCheck{Reason: "unsupported_options"}
		case "saveconfig":
			if !strings.EqualFold(value, "false") {
				return DefaultEgressCheck{Reason: "unsupported_options"}
			}
		case "privatekey", "publickey", "presharedkey":
			decoded, err := base64.StdEncoding.Strict().DecodeString(value)
			valid := err == nil && len(decoded) == 32
			clear(decoded)
			if !valid {
				return DefaultEgressCheck{Reason: "invalid_config"}
			}
		case "listenport", "mtu", "persistentkeepalive":
			if key == "persistentkeepalive" && value == "off" {
				continue
			}
			n, err := strconv.ParseUint(value, 10, 16)
			if err != nil || (key == "mtu" && n < 1280) {
				return DefaultEgressCheck{Reason: "invalid_config"}
			}
		case "endpoint":
			host, port, err := net.SplitHostPort(value)
			n, portErr := strconv.ParseUint(port, 10, 16)
			if err != nil || !validWireGuardEndpointHost(host) || portErr != nil || n == 0 {
				return DefaultEgressCheck{Reason: "missing_endpoint"}
			}
		}
	}

	switch {
	case peers != 1:
		return DefaultEgressCheck{Reason: "peer_count"}
	case !seen["publickey"]:
		return DefaultEgressCheck{Reason: "invalid_config"}
	case !seen["endpoint"]:
		return DefaultEgressCheck{Reason: "missing_endpoint"}
	case !address4 || !address6:
		return DefaultEgressCheck{Reason: "missing_addresses"}
	case !default4 || !default6:
		return DefaultEgressCheck{Reason: "missing_defaults"}
	case !dns:
		return DefaultEgressCheck{Reason: "missing_dns"}
	default:
		return DefaultEgressCheck{Compatible: true}
	}
}

func validWireGuardEndpointHost(host string) bool {
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.IsGlobalUnicast() && addr.Zone() == ""
	}
	host = strings.TrimSuffix(host, ".")
	if len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-", c) {
				return false
			}
		}
	}
	return true
}

// hasAutomaticWireGuardDefault identifies a legacy file that may compete with native
// default routing. ValidateWireGuardConfig must have accepted it first.
func hasAutomaticWireGuardDefault(data []byte) bool {
	var table string
	var hasDefault bool
	lines := bufio.NewScanner(bytes.NewReader(data))
	for lines.Scan() {
		line, _, _ := strings.Cut(lines.Text(), "#")
		key, value, _ := strings.Cut(line, "=")
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "table":
			table = strings.TrimSpace(value)
		case "allowedips":
			for _, item := range strings.Split(value, ",") {
				if prefix, err := netip.ParsePrefix(strings.TrimSpace(item)); err == nil && prefix.Bits() == 0 {
					hasDefault = true
				}
			}
		}
	}
	return hasDefault && (table == "" || table == "auto")
}
