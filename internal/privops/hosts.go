package privops

import (
	"errors"
	"sort"
	"strings"
)

// Hosts block markers. They match internal/monitor/splitdns.go so a block
// written by older builds is replaced, not duplicated.
const (
	HostsStartMarker = "# kongtrol split-dns start"
	HostsEndMarker   = "# kongtrol split-dns end"
)

// ReplaceHostsBlock returns current with the Kongtrol block replaced by
// entries (removed when entries is empty). Lines outside the markers are
// kept byte for byte. Unbalanced or nested markers are an error, so a
// damaged file is never truncated after a lone start marker.
func ReplaceHostsBlock(current string, entries []HostEntry) (string, error) {
	nl := "\n"
	if strings.Contains(current, "\r\n") {
		nl = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(current, "\r\n", "\n"), "\n")
	var kept []string
	in, seen := false, false
	insertAt := -1
	for _, line := range lines {
		switch strings.TrimSpace(line) {
		case HostsStartMarker:
			if in || seen {
				return "", errors.New("hosts: duplicate or nested Kongtrol start marker")
			}
			in, seen = true, true
			insertAt = len(kept)
			continue
		case HostsEndMarker:
			if !in {
				return "", errors.New("hosts: Kongtrol end marker without start marker")
			}
			in = false
			continue
		}
		if !in {
			kept = append(kept, line)
		}
	}
	if in {
		return "", errors.New("hosts: Kongtrol start marker without end marker")
	}
	// Drop trailing empty lines; one newline is re-added below.
	for len(kept) > 0 && kept[len(kept)-1] == "" {
		kept = kept[:len(kept)-1]
		if insertAt > len(kept) {
			insertAt = len(kept)
		}
	}
	block := renderHostsBlock(entries)
	if len(block) > 0 {
		if insertAt < 0 {
			insertAt = len(kept)
		}
		out := append([]string{}, kept[:insertAt]...)
		out = append(out, block...)
		kept = append(out, kept[insertAt:]...)
	}
	return strings.Join(kept, nl) + nl, nil
}

func renderHostsBlock(entries []HostEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	sorted := append([]HostEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Host < sorted[j].Host })
	out := []string{HostsStartMarker}
	for _, e := range sorted {
		out = append(out, e.IP.String()+" "+e.Host)
	}
	return append(out, HostsEndMarker)
}
