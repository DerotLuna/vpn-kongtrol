# Research: Direct diagnostics links

Version 1.0.0 · 2026-11-21

- Existing Kongtrol browser launches exchange a nonce in a URL fragment.
  [ADR-0039](../adr/0039-browser-sessions-one-time-nonce.md) describes the design.
  The query can select the Settings section and coexist with the nonce.
- The flyout's bridge already recognizes fixed authenticated commands. Reuse
  that allowlist for the Diagnostics action. Reading a report uses the same
  diagnostic code in the CLI and panel.
- Apple template images use their alpha channel to let the system choose the
  icon's color. That limits the tray color on macOS. [NSImage.isTemplate](https://developer.apple.com/documentation/appkit/nsimage/istemplate) (accessed 2026-11-21).
