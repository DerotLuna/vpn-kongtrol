# Research: user disconnect reported as a kill switch drop (#46)

Sources checked 2026-11-23. Code read at commit `6ea1f19`.

- Issue #46 (GitHub, DerotLuna/vpn-kongtrol): panel disconnect of an OpenVPN profile on Windows
  showed `killswitch.event.blocking`.
- `internal/app/profile_service.go` `disconnect`: `beginDisconnect` → `MarkIntended` →
  `onIntent(name, true)` → `adapter.Disconnect` → `afterDisconnect` (`applyKill`).
- `internal/app/killswitch_service.go` `Apply`: statuses read before `s.mu`; step 1 set
  `wasUp, intended = true, false` for any connected tunnel unless `disconnecting[name] > 0`.
  `BeginDisconnect` sets that counter only for profiles with rules whose tunnel-down action blocks.
- `cmd/kongtrol/daemon_cmd.go`: `killSwitchSvc.Run(ctx, 5*time.Second)`, so an `Apply` can run
  at any point during `adapter.Disconnect`.
- `internal/monitor/watchdog.go` `watch`: status read first, `paused` checked after it.
  `MarkIntended` precedes `adapter.Disconnect`, so the watchdog never treats it as a drop.
- Every disconnect surface reaches `ProfileService.Disconnect`: `internal/api/handlers_tunnels.go`,
  `internal/api/handlers_groups.go`, `cmd/kongtrol-tray/tray_actions.go` (through the API),
  `cmd/kongtrol/cmd_down.go` and `up_tui.go` (`disconnectProfile`).

Verified by tests that reproduce both orderings (see ADR-0097). Not reproduced on a real Windows
OpenVPN tunnel: **no verificado**.
