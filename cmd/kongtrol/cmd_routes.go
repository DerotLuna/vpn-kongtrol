package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
)

// ── routes ───────────────────────────────────────────────────────────────────

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Manage routing rules",
}

var routesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active Kongtrol-managed routes",
	RunE: func(cmd *cobra.Command, args []string) error {
		routes, err := routeMgr.List()
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				Count  int             `json:"count"`
				Routes []routing.Route `json:"routes"`
			}{
				Count:  len(routes),
				Routes: routes,
			})
		}
		if len(routes) == 0 {
			fmt.Println("  " + styleDim.Render(ct("cli.routes.none")))
			return nil
		}
		destW, gwW, ifaceW, metricW := 24, 20, 14, 6
		for _, r := range routes {
			destW = max(destW, min(42, lipgloss.Width(r.Destination.String())+1))
			if r.Gateway != nil {
				gwW = max(gwW, min(40, lipgloss.Width(r.Gateway.String())+1))
			}
			ifaceW = max(ifaceW, min(24, lipgloss.Width(r.Interface)+1))
		}
		available := terminalWidth() - 2 - 3 // indent + spaces
		sum := destW + gwW + ifaceW + metricW
		if sum > available {
			need := sum - available
			shrinkWidth(&destW, 16, &need)
			shrinkWidth(&gwW, 14, &need)
			shrinkWidth(&ifaceW, 10, &need)
		}
		ruleW := destW + gwW + ifaceW + metricW + 3
		sep := styleDim.Render("  " + strings.Repeat("─", ruleW))
		fmt.Printf("  %s %s %s %s\n",
			styleMapHdr.Render(fitCell(ct("cli.routes.col.dest"), destW)),
			styleMapHdr.Render(fitCell(ct("cli.routes.col.gateway"), gwW)),
			styleMapHdr.Render(fitCell(ct("cli.routes.col.iface"), ifaceW)),
			styleMapHdr.Render(fitCell(ct("cli.routes.col.metric"), metricW)))
		fmt.Println(sep)
		for _, r := range routes {
			gw := "—"
			if r.Gateway != nil {
				gw = r.Gateway.String()
			}
			fmt.Printf("  %s %s %s %s\n",
				styleMapName.Render(fitCell(r.Destination.String(), destW)),
				styleStatusIP.Render(fitCell(gw, gwW)),
				styleMapVia.Render(fitCell(r.Interface, ifaceW)),
				styleStatusTime.Render(fitCell(fmt.Sprintf("%d", r.Metric), metricW)))
		}
		fmt.Println(sep)
		return nil
	},
}

func init() {
	routesCmd.AddCommand(routesListCmd)
}
