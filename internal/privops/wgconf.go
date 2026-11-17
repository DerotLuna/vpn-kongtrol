package privops

import (
	"bufio"
	"bytes"
	"strings"
)

// wgAllowedKeys is the allowlist of WireGuard config keys per section
// (wg-quick + wireguard-windows). PreUp/PostUp/PreDown/PostDown run shell
// commands as root/SYSTEM and are always rejected.
var wgAllowedKeys = map[string]map[string]bool{
	"interface": {"privatekey": true, "listenport": true, "fwmark": true, "address": true,
		"dns": true, "mtu": true, "table": true, "saveconfig": true},
	"peer": {"publickey": true, "presharedkey": true, "allowedips": true, "endpoint": true,
		"persistentkeepalive": true},
}

var wgScriptKeys = map[string]bool{"preup": true, "postup": true, "predown": true, "postdown": true}

// ValidateWireGuardConfig checks a WireGuard .conf: only known sections and
// keys, no hook scripts, exactly one [Interface] with a PrivateKey.
func ValidateWireGuardConfig(content []byte) error {
	if len(content) > MaxConfigSize {
		return invalid("WireGuard config larger than %d bytes", MaxConfigSize)
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return invalid("WireGuard config contains NUL bytes")
	}
	sc := bufio.NewScanner(bytes.NewReader(content))
	section := ""
	interfaces, privKey := 0, false
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if _, ok := wgAllowedKeys[section]; !ok {
				return invalid("line %d: unknown section [%s]", lineNo, section)
			}
			if section == "interface" {
				interfaces++
			}
			continue
		}
		key, _, ok := strings.Cut(line, "=")
		if !ok {
			return invalid("line %d: expected key = value", lineNo)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if wgScriptKeys[key] {
			return invalid("line %d: %s is not allowed (runs commands as administrator)", lineNo, key)
		}
		if section == "" {
			return invalid("line %d: key outside a section", lineNo)
		}
		if !wgAllowedKeys[section][key] {
			return invalid("line %d: key %q is not allowed in [%s]", lineNo, key, section)
		}
		if section == "interface" && key == "privatekey" {
			privKey = true
		}
	}
	if err := sc.Err(); err != nil {
		return invalid("WireGuard config: %v", err)
	}
	if interfaces != 1 || !privKey {
		return invalid("WireGuard config needs exactly one [Interface] with a PrivateKey")
	}
	return nil
}
