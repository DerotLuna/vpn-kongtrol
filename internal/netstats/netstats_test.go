package netstats

import (
	"net"
	"testing"
)

func TestParseNetstatIB(t *testing.T) {
	out := `Name       Mtu   Network       Address            Ipkts Ierrs     Ibytes    Opkts Oerrs     Obytes  Coll
utun4      1420  <Link#21>                          1234     0     567890     4321     0     98765     0
utun4      1420  10.8.0/24     10.8.0.2             1234     -     567890     4321     -     98765     -
`
	sent, recv, ok := parseNetstatIB(out, "utun4")
	if !ok || sent != 98765 || recv != 567890 {
		t.Fatalf("got sent=%d recv=%d ok=%v", sent, recv, ok)
	}
	withAddr := `Name  Mtu   Network       Address            Ipkts Ierrs     Ibytes    Opkts Oerrs     Obytes  Coll
en0   1500  <Link#6>      a4:83:e7:00:00:01  100     0     2000       50     0     1000     0
`
	sent, recv, ok = parseNetstatIB(withAddr, "en0")
	if !ok || sent != 1000 || recv != 2000 {
		t.Fatalf("with address: got sent=%d recv=%d ok=%v", sent, recv, ok)
	}
	if _, _, ok := parseNetstatIB(out, "utun9"); ok {
		t.Fatal("unknown interface must not match")
	}
}

func TestCountersUnknownInterface(t *testing.T) {
	if _, _, ok := Counters(""); ok {
		t.Fatal("empty name must not report counters")
	}
	if _, _, ok := Counters("kongtrol-no-such-if0"); ok {
		t.Fatal("missing interface must not report counters")
	}
}

func TestCountersLoopbackOrFirst(t *testing.T) {
	ifs, err := net.Interfaces()
	if err != nil || len(ifs) == 0 {
		t.Skip("no interfaces")
	}
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 {
			continue
		}
		if _, _, ok := Counters(ifc.Name); ok {
			return
		}
	}
	t.Skip("no interface reported counters on this host")
}
