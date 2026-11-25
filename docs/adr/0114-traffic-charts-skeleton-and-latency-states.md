<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0114: Tráfico charts — same-size skeleton, and latency that never spins forever

Version 1.0.0 · 2026-11-24

Status: Accepted · Date: 2026-11-24 · Decision-makers: Ruben Luna (owner), implemented by agent

## Context and Problem Statement

On Tráfico, each connected tunnel shows Recibido, Enviado and Latencia. Until two samples existed
the card showed "Juntando datos… el gráfico aparece en unos segundos." over a big empty area, and
the layout jumped when the charts arrived. The Latencia chart showed "—" with a spinner for as
long as no reading came, which for some OpenVPN profiles was forever.

Why no reading (verified in code, 2026-11-24): latency comes only from the daemon's health probe
(`healthCheckProfile`, `cmd/kongtrol/daemon_runtime.go`): a TCP handshake with `vpns.<name>.host`
(port `port`, else 443) or the tunnel's `RemoteIP`, else a TCP handshake with / lookup through the
tunnel's DNS. The OpenVPN adapter reports no `RemoteIP` and its config usually has no `host`, so
it depends on the pushed DNS seen in the helper log; a UDP-only server that pushes no DNS (or a
DNS line Kongtrol did not see) gives no round trip at all. Also, the panel records a latency point
only when the value changes, so a steady latency stayed at one point and kept the spinner.

Measuring OpenVPN like WireGuard was considered: reporting the server address from the management
`state` line would make the probe dial a UDP-only server over TCP, which fails or waits until the
probe timeout and would turn the health check into a false "degraded" notice. Not done; **no
verificado** on a real UDP-only OpenVPN server with this machine's helper.

## Decision

1. **Skeleton.** Until two samples exist the card shows a skeleton of the three charts with the
   same boxes as the real ones (head, 72 px plot, x axis, two summary lines), built from a new kit
   primitive `SkeletonBlock` (`src/components/ui/States.tsx`, shown on `/_kit`): shimmer that
   stops under `prefers-reduced-motion`, `aria-busy="true"` on the region and a visually hidden
   "Cargando gráficos" status for screen readers.
2. **Latency states.** Two or more readings: the chart. One reading: its value and "Última
   medición; el gráfico aparece cuando cambie." (no spinner). None: a loader only while a first
   reading is still expected — `LATENCY_WAIT_MS` = 45 s of samples (the daemon probes within one
   5 s watchdog pass, 10 s default probe timeout) — then a static "Sin medición para esta VPN" with
   an info button whose kit Tooltip says how latency is measured and that the connection is not
   affected. A later reading still draws the chart.

## Consequences

- Good: nothing jumps when the charts arrive; no endless spinner; the user learns why there is no
  number.
- Bad: OpenVPN profiles without a probe target still have no latency; a real per-tunnel probe
  (e.g. ICMP or the server's own protocol) would need its own decision.
