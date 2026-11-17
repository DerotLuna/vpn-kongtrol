package openvpn

import (
	"reflect"
	"testing"
)

func TestParsePushedRoutes(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		routes   []string
		redirect bool
		cont     bool
	}{
		{
			name:   "routes and hosts",
			line:   "PUSH: Received control message: 'PUSH_REPLY,route 10.200.0.0 255.255.255.0,route 192.168.220.0 255.255.255.0,route 200.71.242.80 255.255.255.255,route 10.1.131.31,topology net30,ping 10,ifconfig 192.168.220.250 192.168.220.249'",
			routes: []string{"10.200.0.0/24", "192.168.220.0/24", "200.71.242.80/32", "10.1.131.31/32"},
		},
		{
			name:   "route-ipv6",
			line:   "PUSH_REPLY,route-ipv6 2001:db8::/32 2001:db8::1,route-ipv6 fd00::1,ifconfig-ipv6 fd00::2/64 fd00::1",
			routes: []string{"2001:db8::/32", "fd00::1/128"},
		},
		{
			name:     "redirect-gateway def1",
			line:     "PUSH_REPLY,redirect-gateway def1 bypass-dhcp,dhcp-option DNS 10.8.0.1",
			redirect: true,
		},
		{
			name:   "redirect-gateway ipv6 !ipv4",
			line:   "PUSH_REPLY,redirect-gateway ipv6 !ipv4,route 10.0.0.0 255.0.0.0",
			routes: []string{"::/0", "10.0.0.0/8"},
		},
		{
			name:   "hostname, bad mask and continuation",
			line:   "PUSH_REPLY,route vpn.example.com,route 10.0.0.0 255.0.255.0,route 10.9.0.0 255.255.0.0 vpn_gateway 5,push-continuation 2",
			routes: []string{"10.9.0.0/16"},
			cont:   true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, ok := parsePushedRoutes(c.line)
			if !ok {
				t.Fatal("not recognized as PUSH_REPLY")
			}
			if !reflect.DeepEqual(p.routes, c.routes) || p.redirect4 != c.redirect || p.cont != c.cont {
				t.Fatalf("got %+v", p)
			}
		})
	}
	if _, ok := parsePushedRoutes("Initialization Sequence Completed"); ok {
		t.Fatal("non-PUSH_REPLY line parsed")
	}
}

func TestLogStatePushedRoutes(t *testing.T) {
	s := &logState{}
	s.consume("PUSH_REPLY,route 10.1.0.0 255.255.0.0,push-continuation 2")
	s.consume("PUSH_REPLY,route 10.2.0.0 255.255.0.0,push-continuation 1")
	routes, redirect, ok := s.pushed()
	if !ok || redirect || !reflect.DeepEqual(routes, []string{"10.1.0.0/16", "10.2.0.0/16"}) {
		t.Fatalf("continuation not merged: %v %v %v", routes, redirect, ok)
	}
	// A later PUSH_REPLY (reconnect) replaces the previous set.
	s.consume("PUSH_REPLY,redirect-gateway def1")
	routes, redirect, ok = s.pushed()
	if !ok || !redirect || len(routes) != 0 {
		t.Fatalf("new PUSH_REPLY did not replace: %v %v %v", routes, redirect, ok)
	}

	// Helper: PushReply wins over the tail, and survives the tail scrolling.
	h := &logState{}
	push := []string{"PUSH_REPLY,route 10.3.0.0 255.255.0.0"}
	h.replaceTail([]string{"PUSH_REPLY,route 10.9.9.0 255.255.255.0"}, push)
	h.replaceTail([]string{"Initialization Sequence Completed"}, push)
	if routes, _, _ := h.pushed(); !reflect.DeepEqual(routes, []string{"10.3.0.0/16"}) {
		t.Fatalf("helper push reply: %v", routes)
	}
	// Old helper (no PushReply): routes come from the tail.
	o := &logState{}
	o.replaceTail([]string{"PUSH_REPLY,route 10.4.0.0 255.255.0.0"}, nil)
	o.replaceTail([]string{"later line"}, nil)
	if routes, _, _ := o.pushed(); !reflect.DeepEqual(routes, []string{"10.4.0.0/16"}) {
		t.Fatalf("tail push reply: %v", routes)
	}
}
