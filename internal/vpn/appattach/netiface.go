package appattach

import "net"

// netInterfaces lists interfaces through the portable net package (names
// only, no descriptions).
func netInterfaces() []Iface {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := make([]Iface, 0, len(ifaces))
	for _, i := range ifaces {
		ifc := Iface{Name: i.Name, Up: i.Flags&net.FlagUp != 0}
		if addrs, err := i.Addrs(); err == nil {
			for _, a := range addrs {
				if ipn, ok := a.(*net.IPNet); ok {
					ifc.Addrs = append(ifc.Addrs, ipn.IP)
				}
			}
		}
		out = append(out, ifc)
	}
	return out
}
