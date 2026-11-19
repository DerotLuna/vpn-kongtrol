package policy

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Targets is what a list of free-form entries ("github.com",
// "https://intranet.example.com/x", "10.0.0.0/8", "app:chrome.exe") means
// as a policy match: the input of the "＋ Agregar" flow and of
// `kongtrol policy add`.
type Targets struct {
	Domains  []string
	IPRanges []string
	Apps     []string
	// First is the first entry in its stored form (host, IP, range or
	// app), used to name the policy.
	First string
}

// domainPattern is the same check as web/ui isDomainPattern: exact names
// and one leading "*." label, the only forms the engine matches (matchGlob).
var domainPattern = regexp.MustCompile(`^(?:\*\.)?(?:[a-z0-9_](?:[a-z0-9_-]{0,61}[a-z0-9_])?\.)*[a-z0-9_](?:[a-z0-9_-]{0,61}[a-z0-9_])?$`)

// ClassifyTargets sorts entries into domains, IP ranges and apps, the way
// the dashboard's "＋ Agregar" flow does (web/ui mirrors these rules; keep
// both in step):
//   - "app:<exe or path>" → apps;
//   - an IP or a CIDR → ip_ranges (a bare IP stays bare; the engine adds /32);
//   - a URL or host[:port][/path] → its host, lowercased, as a domain, plus
//     "*.<host>" for its subdomains unless subdomains is false (a "*."
//     entry is kept as written).
//
// Duplicates are dropped, order is kept. invalid lists the entries that are
// none of these, as written.
func ClassifyTargets(entries []string, subdomains bool) (t Targets, invalid []string) {
	seen := map[string]bool{}
	add := func(list *[]string, v string) {
		if !seen[v] {
			seen[v] = true
			*list = append(*list, v)
		}
	}
	for _, raw := range entries {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if len(s) >= 4 && strings.EqualFold(s[:4], "app:") {
			app := strings.TrimSpace(s[4:])
			if app == "" {
				invalid = append(invalid, raw)
				continue
			}
			add(&t.Apps, app)
			t.setFirst(app)
			continue
		}
		if r, ok := ipOrCIDR(s); ok {
			add(&t.IPRanges, r)
			t.setFirst(r)
			continue
		}
		if i := strings.IndexByte(s, '/'); i > 0 && !strings.Contains(s, "://") && net.ParseIP(s[:i]) != nil {
			invalid = append(invalid, raw) // an IP with a bad prefix length
			continue
		}
		host := hostOf(s)
		if ip := net.ParseIP(host); ip != nil {
			add(&t.IPRanges, ip.String())
			t.setFirst(ip.String())
			continue
		}
		if host == "" || len(host) > 253 || !domainPattern.MatchString(host) {
			invalid = append(invalid, raw)
			continue
		}
		add(&t.Domains, host)
		t.setFirst(host)
		if subdomains && !strings.HasPrefix(host, "*.") {
			add(&t.Domains, "*."+host)
		}
	}
	return t, invalid
}

func (t *Targets) setFirst(v string) {
	if t.First == "" {
		t.First = v
	}
}

// ipOrCIDR accepts what policy.ParseRule accepts as an ip_range: a bare IP
// or a CIDR (net.ParseCIDR).
func ipOrCIDR(s string) (string, bool) {
	if ip := net.ParseIP(s); ip != nil {
		return ip.String(), true
	}
	if strings.Contains(s, "/") {
		if _, _, err := net.ParseCIDR(s); err == nil {
			return s, true
		}
	}
	return "", false
}

// hostOf returns the bare host of a URL or host[:port][/path], lowercased,
// without a trailing dot (as handleExplainPolicy normalizes a target).
func hostOf(s string) string {
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Hostname() == "" {
			return ""
		}
		s = u.Hostname()
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		s = h
	}
	s = strings.Trim(s, "[]")
	s = strings.TrimSuffix(s, ".")
	return strings.ToLower(strings.TrimSpace(s))
}

// AutoName is the name a new policy gets when none is given: format
// (translated, e.g. "%[1]s por %[2]s") filled with the first entry (an
// app's file name, not its folder) and the VPN, made a valid name
// (CleanName) and unique among existing (SuggestName).
func AutoName(format, first, via string, existing []string) string {
	if i := strings.LastIndexAny(first, `/\`); i >= 0 && i < len(first)-1 {
		first = first[i+1:]
	}
	return SuggestName(CleanName(fmt.Sprintf(format, first, via), 60), existing)
}

// CleanName drops the characters config.ValidateName refuses in a name
// (< > " ' ` & \, control and line/paragraph separators) and keeps at most
// max characters.
func CleanName(s string, max int) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= max {
			break
		}
		if unicode.IsControl(r) || r == ' ' || r == ' ' || strings.ContainsRune("<>\"'`&\\", r) {
			continue
		}
		b.WriteRune(r)
		n++
	}
	return strings.TrimSpace(b.String())
}

// SuggestName returns base, or base followed by " 2", " 3", … when a
// policy with that name (case-insensitive, like the daemon) already exists.
func SuggestName(base string, existing []string) string {
	taken := map[string]bool{}
	for _, n := range existing {
		taken[strings.ToLower(n)] = true
	}
	if !taken[strings.ToLower(base)] {
		return base
	}
	for i := 2; ; i++ {
		n := base + " " + strconv.Itoa(i)
		if !taken[strings.ToLower(n)] {
			return n
		}
	}
}
