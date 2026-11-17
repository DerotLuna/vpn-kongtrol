package vpn

// RouteOrigin says why a destination goes through a tunnel.
type RouteOrigin string

const (
	// OriginServer: pushed by the VPN server (OpenVPN PUSH_REPLY route /
	// route-ipv6 / redirect-gateway).
	OriginServer RouteOrigin = "server"
	// OriginPolicy: installed by Kongtrol for a policy rule.
	OriginPolicy RouteOrigin = "policy"
	// OriginVPN: the VPN's own configuration (WireGuard AllowedIPs, routes
	// in the client config) or any other OS route on the tunnel interface.
	OriginVPN RouteOrigin = "vpn"
)

// TunnelRoutes is what an adapter knows about the destinations it sends
// through its tunnel, beyond what the OS route table shows.
type TunnelRoutes struct {
	Origin RouteOrigin
	// Routes are CIDRs ("10.200.0.0/24", "2001:db8::/32").
	Routes []string
	// AllTraffic is true when the tunnel takes the IPv4 default route
	// (OpenVPN redirect-gateway, WireGuard AllowedIPs 0.0.0.0/0).
	AllTraffic bool
}

// RouteReporter is implemented by adapters that know their tunnel's routes
// (OpenVPN from the server's PUSH_REPLY, WireGuard from AllowedIPs).
// TunnelRoutes returns nil when there is nothing to report.
type RouteReporter interface {
	TunnelRoutes() *TunnelRoutes
}
