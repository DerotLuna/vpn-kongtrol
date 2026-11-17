package privops

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"strings"
)

// ovpnDenied lists directives that run programs, load modules, write files
// as the privileged user, include other files or take over the management
// interface. A config containing any of them is rejected.
var ovpnDenied = map[string]bool{
	// scripts and hooks
	"up": true, "down": true, "route-up": true, "route-pre-down": true, "ipchange": true,
	"client-connect": true, "client-disconnect": true, "learn-address": true,
	"auth-user-pass-verify": true, "tls-verify": true, "tls-export-cert": true,
	"tls-crypt-v2-verify": true, "plugin": true,
	// executables / modules picked by path
	"iproute": true, "win-sys": true, "engine": true, "providers": true,
	"pkcs11-providers": true, "pkcs11-id-management": true,
	// files written or directories changed as the privileged user
	"log": true, "log-append": true, "status": true, "writepid": true,
	"replay-persist": true, "ifconfig-pool-persist": true, "tmp-dir": true,
	"cd": true, "chroot": true, "client-config-dir": true, "auth-gen-token-secret": true,
	// process control, includes, management (Kongtrol owns these)
	"config": true, "daemon": true, "msg-channel": true, "service": true,
	"capath": true, "askpass": true,
}

// ovpnFileRefs are directives whose first argument is a file the helper
// reads itself (with the caller's access) and inlines, so openvpn never
// opens a user path.
var ovpnFileRefs = map[string]bool{
	"ca": true, "cert": true, "key": true, "tls-auth": true, "tls-crypt": true,
	"tls-crypt-v2": true, "secret": true, "dh": true, "extra-certs": true,
	"crl-verify": true, "pkcs12": true,
}

// ovpnInlineTags are the <tag> blocks allowed in a config.
var ovpnInlineTags = map[string]bool{
	"ca": true, "cert": true, "key": true, "tls-auth": true, "tls-crypt": true,
	"tls-crypt-v2": true, "secret": true, "dh": true, "extra-certs": true,
	"crl-verify": true, "pkcs12": true, "auth-user-pass": true,
	"http-proxy-user-pass": true, "connection": true,
}

// RefReader reads a file referenced from a config. path is exactly as
// written in the config (may be relative to the config's directory).
type RefReader func(path string) ([]byte, error)

// SanitizeOpenVPNConfig validates an OpenVPN config and returns a
// self-contained copy: denied directives are rejected, referenced key/cert
// files are inlined through read, and a script-security above 1 is refused.
// The caller still appends --script-security 1 on the command line.
func SanitizeOpenVPNConfig(content []byte, read RefReader) ([]byte, error) {
	if len(content) > MaxConfigSize {
		return nil, invalid("OpenVPN config larger than %d bytes", MaxConfigSize)
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return nil, invalid("OpenVPN config contains NUL bytes")
	}
	var out bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 64<<10), MaxConfigSize)
	inTag := ""
	lineNo := 0
	for sc.Scan() {
		lineNo++
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		if inTag != "" && inTag != "connection" {
			if strings.EqualFold(line, "</"+inTag+">") {
				inTag = ""
			}
			out.WriteString(raw + "\n")
			continue
		}
		if line == "" || line[0] == '#' || line[0] == ';' {
			out.WriteString(raw + "\n")
			continue
		}
		if strings.HasPrefix(line, "</") && strings.HasSuffix(line, ">") {
			tag := strings.ToLower(line[2 : len(line)-1])
			if tag != inTag {
				return nil, invalid("line %d: unexpected %s", lineNo, line)
			}
			inTag = ""
			out.WriteString(raw + "\n")
			continue
		}
		if strings.HasPrefix(line, "<") && strings.HasSuffix(line, ">") {
			tag := strings.ToLower(line[1 : len(line)-1])
			if !ovpnInlineTags[tag] {
				return nil, invalid("line %d: inline block <%s> is not allowed", lineNo, tag)
			}
			if inTag == "connection" && tag == "connection" {
				return nil, invalid("line %d: nested <connection>", lineNo)
			}
			if inTag == "connection" {
				// Inline file blocks inside <connection> are not valid OpenVPN.
				return nil, invalid("line %d: <%s> inside <connection>", lineNo, tag)
			}
			inTag = tag
			out.WriteString(raw + "\n")
			continue
		}
		args, err := splitOVPNArgs(line)
		if err != nil {
			return nil, invalid("line %d: %v", lineNo, err)
		}
		if len(args) == 0 {
			continue
		}
		name := strings.ToLower(strings.TrimPrefix(args[0], "--"))
		repl, err := checkOVPNDirective(name, args[1:], read, lineNo)
		if err != nil {
			return nil, err
		}
		if repl != "" {
			out.WriteString(repl)
		} else {
			out.WriteString(raw + "\n")
		}
	}
	if err := sc.Err(); err != nil {
		return nil, invalid("OpenVPN config: %v", err)
	}
	if inTag != "" {
		return nil, invalid("inline block <%s> is not closed", inTag)
	}
	return out.Bytes(), nil
}

func checkOVPNDirective(name string, args []string, read RefReader, lineNo int) (string, error) {
	switch {
	case ovpnDenied[name], strings.HasPrefix(name, "management"):
		return "", invalid("line %d: directive %q is not allowed (runs scripts, writes files or is managed by Kongtrol)", lineNo, name)
	case name == "script-security":
		if len(args) == 0 || (args[0] != "0" && args[0] != "1") {
			return "", invalid("line %d: script-security above 1 is not allowed", lineNo)
		}
	case name == "auth-user-pass":
		if len(args) > 0 {
			return "", invalid("line %d: auth-user-pass must not name a file (Kongtrol sends credentials)", lineNo)
		}
	case name == "http-proxy" || name == "socks-proxy":
		if len(args) >= 3 && !(name == "http-proxy" && (args[2] == "auto" || args[2] == "auto-nct")) {
			return "", invalid("line %d: %s must not name a credentials file", lineNo, name)
		}
	case ovpnFileRefs[name]:
		if len(args) == 0 {
			return "", invalid("line %d: %s needs a file", lineNo, name)
		}
		if args[0] == "[inline]" {
			return "", nil
		}
		if name == "crl-verify" && len(args) > 1 {
			return "", invalid("line %d: crl-verify directory mode is not allowed", lineNo)
		}
		if read == nil {
			return "", invalid("line %d: %s references a file", lineNo, name)
		}
		data, err := read(args[0])
		if err != nil {
			return "", err
		}
		return inlineBlock(name, data, args[1:])
	}
	return "", nil
}

// inlineBlock renders <name>data</name> plus key-direction when given.
func inlineBlock(name string, data []byte, rest []string) (string, error) {
	if len(data) > MaxReferenceSize {
		return "", invalid("%s file larger than %d bytes", name, MaxReferenceSize)
	}
	body := string(data)
	if name == "pkcs12" {
		body = base64.StdEncoding.EncodeToString(data)
	} else if bytes.IndexByte(data, 0) >= 0 || strings.Contains(strings.ToLower(body), "</") {
		return "", invalid("%s file is not a PEM/text key file", name)
	}
	var b strings.Builder
	b.WriteString("<" + name + ">\n")
	b.WriteString(strings.TrimRight(body, "\r\n"))
	b.WriteString("\n</" + name + ">\n")
	if (name == "tls-auth" || name == "secret") && len(rest) > 0 {
		if rest[0] != "0" && rest[0] != "1" {
			return "", invalid("%s key direction must be 0 or 1", name)
		}
		b.WriteString("key-direction " + rest[0] + "\n")
	}
	return b.String(), nil
}

// splitOVPNArgs splits a directive line like OpenVPN does: whitespace
// separated, double or single quotes group, backslash escapes in double
// quotes, a '#' or ';' starting a token begins a comment.
func splitOVPNArgs(line string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inTok, quote := false, rune(0)
	escape := false
	for _, r := range line {
		switch {
		case escape:
			cur.WriteRune(r)
			escape = false
		case quote != 0:
			if r == '\\' && quote == '"' {
				escape = true
			} else if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, inTok = r, true
		case r == '\\':
			escape, inTok = true, true
		case r == ' ' || r == '\t':
			if inTok {
				args = append(args, cur.String())
				cur.Reset()
				inTok = false
			}
		case (r == '#' || r == ';') && !inTok:
			return args, nil
		default:
			cur.WriteRune(r)
			inTok = true
		}
	}
	if quote != 0 || escape {
		return nil, errUnterminated
	}
	if inTok {
		args = append(args, cur.String())
	}
	return args, nil
}

var errUnterminated = invalid("unterminated quote")
