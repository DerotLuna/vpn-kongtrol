package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/uninstall"
	"github.com/vpn-kongtrol/kongtrol/internal/update"
)

// Opt-in update notice (ADR-0034) and the dashboard's "Borrar todos mis
// datos" (ADR-0034), wired into the daemon.

var (
	updateChecker = update.New(version, func() bool { return cfg.UpdateCheckEnabled() })
	updateKick    = make(chan struct{}, 1)
)

// startUpdateNotice runs the checker for the daemon's lifetime. With
// updates.check_enabled false it never makes a request.
func startUpdateNotice(ctx context.Context) {
	go updateChecker.Run(ctx, func(r update.Result) {
		url := r.URL
		if url == "" {
			url = update.DownloadPage
		}
		daemonEvents.Add(monitor.Event{
			Kind:  monitor.EventUpdateAvailable,
			Level: "INFO",
			Key:   "update.event.available",
			Args:  []string{r.Latest, r.Current, url},
		})
	}, updateKick)
}

// kickUpdateCheck checks right away after the user turned the notice on.
func kickUpdateCheck(on bool) {
	if !on {
		return
	}
	select {
	case updateKick <- struct{}{}:
	default:
	}
}

// wireLifecycleAPI enables /api/v1/updates and /api/v1/data/purge.
func wireLifecycleAPI(srv *api.Server) {
	srv.SetUpdateChecker(updateChecker.Last, kickUpdateCheck)
	srv.SetDataPurger(purgeDataFromDaemon)
}

// kongtrolDataDir is ~/.kongtrol.
func kongtrolDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kongtrol")
}

// purgeExtraFiles are user files Kongtrol keeps outside ~/.kongtrol: a
// --config file elsewhere and the daily files of a custom audit log path
// (security.AuditLogFiles; never the path itself, which may be anything).
func purgeExtraFiles(c *config.Config) []string {
	extra := []string{activeCfgPath}
	if c != nil && c.Security.AuditLog.Path != "" {
		extra = append(extra, security.AuditLogFiles(c.Security.AuditLog.Path)...)
	}
	return extra
}

// purgeDataFromDaemon disconnects every tunnel (lifting kill switch blocks
// and restoring DNS, so the user stays online), then deletes settings,
// profiles and Kongtrol's keychain entries. The files this running daemon
// still needs (api-token, run/, logs) stay; the caller restarts it into
// first-run setup.
func purgeDataFromDaemon() error {
	disconnectActiveOnShutdown(adapters.Snapshot(), disconnectProfile, shutdownDisconnectTimeout)
	if destBlocker != nil {
		_ = destBlocker.Set(nil)
	}
	if ks != nil {
		_ = ks.Disable()
	}
	if dnsMgr != nil {
		dnsMgr.ForceRestore()
	}
	extra := purgeExtraFiles(cfg)
	closeAudit()
	rep := uninstall.Purge(uninstall.PurgeOptions{
		DataDir:     kongtrolDataDir(),
		ExtraFiles:  extra,
		KeepRuntime: true,
	})
	fmt.Fprintln(os.Stderr, tuiInfo(cf("cli.uninstall.purged", len(rep.Credentials))))
	return rep.Err()
}

// updateLine is the "Versión nueva disponible" line for `kongtrol version`
// and `status`, from the remembered last check (no request). Empty when the
// notice is off or there is nothing newer.
func updateLine() string {
	c := cfg
	if c == nil {
		var err error
		if c, err = config.Load(cfgPath); err != nil {
			return ""
		}
	}
	if !c.UpdateCheckEnabled() {
		return ""
	}
	r := update.LastFrom(version, update.DefaultStatePath())
	if !r.Newer {
		return ""
	}
	return cf("cli.update.available", r.Latest, r.Current)
}
