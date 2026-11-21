# ADR-0083: Open Diagnostics from a connection error

Version 1.0.0 · 2026-11-21

Status: Accepted · Date: 2026-11-21

Use one Diagnostics destination in Settings for the dashboard, tray and CLI.
The tray's native menu and flyout open that destination using the same
authenticated panel launch as other pages. The CLI supports
`kongtrol dashboard --diagnostics`; `kongtrol doctor` remains the text report.

Show VPN connection errors with the existing red danger color. Keep transient
warnings in the caution state. macOS uses the system template tray icon; the OS
renders its icon in monochrome.

[Investigation](../research/2026-11-21-diagnostics-deep-links.md)
