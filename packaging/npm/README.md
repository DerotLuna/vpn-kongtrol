<img src="https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# kongtrol (npm)

Version 1.1.0 · 2026-11-18

Kongtrol CLI from npm: connect and control several VPNs from one place, with a local
dashboard in your browser. Nothing leaves your machine.

```bash
npx kongtrol init          # set up your first VPN
npm i -g kongtrol          # or: pnpm add -g kongtrol · yarn global add kongtrol · bun add -g kongtrol
kongtrol up                # connect; `kongtrol dashboard` opens the panel
npm update -g kongtrol     # upgrade; then `kongtrol restart` keeps your tunnels
```

Your settings live in `~/.kongtrol` and are kept on upgrade and uninstall (to delete them, run
`kongtrol uninstall --purge` before `npm uninstall -g kongtrol`). The tray app
and the installers (Windows, macOS, Linux) are on https://kongtrol.app.

Español: `npx kongtrol init` configura tu primera VPN; `npm update -g kongtrol` actualiza
y `kongtrol restart` aplica la nueva versión sin perder tus túneles. Guía: https://kongtrol.app/guia
