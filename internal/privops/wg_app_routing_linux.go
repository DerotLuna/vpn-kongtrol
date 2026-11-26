package privops

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func supportsWGAppRouting() bool {
	var fs unix.Statfs_t
	return strconv.IntSize == 64 && unix.Statfs("/sys/fs/cgroup", &fs) == nil && fs.Type == unix.CGROUP2_SUPER_MAGIC
}

func (l *Local) startWGAppRouting(name string, uid int, req WGAppRouting) (_ wgAppRoutingSession, err error) {
	if uid < 0 {
		return wgAppRoutingSession{}, ErrDenied
	}
	egress, err := netlink.LinkByName(name)
	if err != nil {
		return wgAppRoutingSession{}, err
	}
	if egress.Type() != "wireguard" {
		return wgAppRoutingSession{}, errors.New("selected interface is not WireGuard")
	}
	// Locking is already held by WGInstallTunnel. Keep the executable file
	// open while marked to prevent its inode from being recycled.
	var opened []*os.File
	keys := map[[3]uint64]struct{}{}
	var exeKeys []appSocketKey
	var tables []*appRouteTable
	var classifier *appSocketClassifier
	closeSession := func() error {
		var errs []error
		if classifier != nil {
			errs = append(errs, classifier.Close())
			classifier = nil
		}
		for _, table := range tables {
			errs = append(errs, table.Close())
		}
		for _, file := range opened {
			errs = append(errs, file.Close())
		}
		opened = nil
		return errors.Join(errs...)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, closeSession())
		}
	}()
	for _, path := range req.Executables {
		fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
		if err != nil {
			return wgAppRoutingSession{}, err
		}
		file := os.NewFile(uintptr(fd), path)
		opened = append(opened, file)
		var st unix.Stat_t
		if err := unix.Fstat(fd, &st); err != nil {
			return wgAppRoutingSession{}, err
		}
		if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Mode&0111 == 0 {
			return wgAppRoutingSession{}, invalid("application is not an executable file")
		}
		device := unix.Major(st.Dev)<<20 | unix.Minor(st.Dev)
		identity := [3]uint64{st.Ino, uint64(device), uint64(uid)}
		if _, duplicate := keys[identity]; duplicate {
			continue
		}
		for _, live := range l.wgAppRouting {
			if _, duplicate := live.keys[identity]; duplicate {
				return wgAppRoutingSession{}, invalid("executable is already assigned to another active profile")
			}
		}
		keys[identity] = struct{}{}
		exeKeys = append(exeKeys, appSocketKey{inode: st.Ino, device: device, uid: uint32(uid)})
	}

	table, priority, err := nextWGAppTable()
	if err != nil {
		return wgAppRoutingSession{}, err
	}
	for _, cidr := range []string{"0.0.0.0/0", "::/0"} {
		route, err := openAppRouteTableWithMode(egress, netip.MustParsePrefix(cidr), table, priority, uint32(table), req.Mode)
		if err != nil {
			return wgAppRoutingSession{}, err
		}
		tables = append(tables, route)
	}
	classifier, err = openAppSocketClassifier("/sys/fs/cgroup")
	if err != nil {
		return wgAppRoutingSession{}, err
	}
	for _, key := range exeKeys {
		if err := classifier.marks.Put(key, uint32(table)); err != nil {
			return wgAppRoutingSession{}, err
		}
	}
	return wgAppRoutingSession{close: closeSession, keys: keys}, nil
}

// Reserve the same mark and table for both address families. Never
// replace user routes or rules, even if an earlier helper left state.
func nextWGAppTable() (table, priority int, err error) {
	const firstTable, firstPriority, slots = 0x4b010001, 100, 128
	usedTables, usedPriorities := map[int]bool{}, map[int]bool{}
	for _, family := range []int{unix.AF_INET, unix.AF_INET6} {
		rules, err := netlink.RuleList(family)
		if err != nil {
			return 0, 0, err
		}
		for _, rule := range rules {
			usedTables[rule.Table] = true
			usedTables[int(rule.Mark)] = true
			usedPriorities[rule.Priority] = true
			if rule.Priority > 0 && rule.Priority < firstPriority+2*slots {
				// Only a rule carrying one of our distinct application marks
				// can precede this connection without overriding its lookup.
				if rule.Mark < firstTable || rule.Mark >= firstTable+slots ||
					(rule.Table != int(rule.Mark) && rule.Table != unix.RT_TABLE_MAIN) {
					return 0, 0, fmt.Errorf("an earlier Linux routing rule may override executable routing (priority %d)", rule.Priority)
				}
			}
		}
	}
	for i := range slots {
		table, priority := firstTable+i, firstPriority+1+2*i
		if usedTables[table] || usedPriorities[priority] || usedPriorities[priority-1] {
			continue
		}
		empty := true
		for _, family := range []int{unix.AF_INET, unix.AF_INET6} {
			routes, err := netlink.RouteListFiltered(family, &netlink.Route{Table: table}, netlink.RT_FILTER_TABLE)
			if err != nil && !errors.Is(err, unix.ENOENT) {
				return 0, 0, err
			}
			empty = empty && len(routes) == 0
		}
		if empty {
			return table, priority, nil
		}
	}
	return 0, 0, errors.New("no free Linux routing table for executable routing")
}
