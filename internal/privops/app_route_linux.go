package privops

import (
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// appRouteTable owns the route and policy rule for a marked application
// egress. The caller supplies a reserved mark, table and priority for the
// network namespace. Use a dedicated table so unmarked sockets skip it.
type appRouteTable struct {
	route           netlink.Route
	fallback        netlink.Route
	rule            netlink.Rule
	destinationRule *netlink.Rule
}

func openAppRouteTable(egress netlink.Link, destination netip.Prefix, table, priority int, mark uint32) (*appRouteTable, error) {
	return openAppRouteTableWithMode(egress, destination, table, priority, mark, AppRouteOverride)
}

func openAppRouteTableWithMode(egress netlink.Link, destination netip.Prefix, table, priority int, mark uint32, mode string) (*appRouteTable, error) {
	if !destination.IsValid() || table <= 0 || priority <= 0 || mark == 0 {
		return nil, errors.New("app routing needs a destination, table, policy priority and nonzero mark")
	}
	family := unix.AF_INET
	if destination.Addr().Is6() {
		family = unix.AF_INET6
	}
	destination = destination.Masked()
	subnet := &net.IPNet{IP: destination.Addr().AsSlice(), Mask: net.CIDRMask(destination.Bits(), destination.Addr().BitLen())}
	rule := netlink.NewRule()
	rule.Family, rule.Table, rule.Priority = family, table, priority
	rule.Mark, rule.Mask = int(mark), -1 // full mark
	rt := &appRouteTable{
		route: netlink.Route{
			LinkIndex: egress.Attrs().Index, Dst: subnet, Table: table, Scope: netlink.SCOPE_LINK,
		},
		// A tunnel interface disappearing removes the unicast route. Keep a
		// terminal lookup in the same table to prevent a fallthrough to main.
		fallback: netlink.Route{
			Dst: subnet, Table: table, Type: unix.RTN_UNREACHABLE, Priority: 32760,
		},
		rule: *rule,
	}
	if err := netlink.RouteAdd(&rt.fallback); err != nil {
		return nil, fmt.Errorf("add app route fallback: %w", err)
	}
	if err := netlink.RouteAdd(&rt.route); err != nil {
		return nil, errors.Join(fmt.Errorf("add app route: %w", err), deleteAppRoute(&rt.fallback))
	}
	if err := netlink.RuleAdd(&rt.rule); err != nil {
		return nil, errors.Join(fmt.Errorf("add app routing rule: %w", err), deleteAppRoute(&rt.route), deleteAppRoute(&rt.fallback))
	}
	if mode == AppRouteFallback {
		destinations := netlink.NewRule()
		destinations.Family, destinations.Table, destinations.Priority = family, unix.RT_TABLE_MAIN, priority-1
		destinations.Mark, destinations.Mask = int(mark), -1
		// Keep explicit routes and LAN routes. A regular default and the
		// two /1 defaults that VPN clients use do not override this app.
		destinations.SuppressPrefixlen = 1
		if err := netlink.RuleAdd(destinations); err != nil {
			return nil, errors.Join(fmt.Errorf("add destination precedence rule: %w", err), rt.Close())
		}
		rt.destinationRule = destinations
	}
	return rt, nil
}

func (rt *appRouteTable) Close() error {
	if rt.destinationRule != nil {
		if err := netlink.RuleDel(rt.destinationRule); err != nil && !errors.Is(err, unix.ENOENT) && !errors.Is(err, unix.ESRCH) {
			return err
		}
		rt.destinationRule = nil
	}
	// Stop the lookup before changing the contents of its dedicated table.
	if err := netlink.RuleDel(&rt.rule); err != nil && !errors.Is(err, unix.ENOENT) && !errors.Is(err, unix.ESRCH) {
		return err
	}
	return errors.Join(deleteAppRoute(&rt.route), deleteAppRoute(&rt.fallback))
}

func deleteAppRoute(route *netlink.Route) error {
	err := netlink.RouteDel(route)
	if errors.Is(err, unix.ESRCH) || errors.Is(err, unix.ENODEV) {
		return nil
	}
	return err
}
