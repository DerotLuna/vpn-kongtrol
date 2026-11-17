package monitor

import "testing"

func TestSplitDNS_HostEntries(t *testing.T) {
	got := hostEntries(map[string]string{"www.Example.com": "1.1.1.1", "api.example.com": "::ffff:2.2.2.2", "bad.example.com": "nope"})
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Host != "api.example.com" || got[0].IP.String() != "2.2.2.2" {
		t.Fatalf("first entry %+v", got[0])
	}
	if got[1].Host != "www.example.com" || got[1].IP.String() != "1.1.1.1" {
		t.Fatalf("second entry %+v", got[1])
	}
	if len(hostEntries(nil)) != 0 {
		t.Fatal("empty map must give no entries")
	}
}
