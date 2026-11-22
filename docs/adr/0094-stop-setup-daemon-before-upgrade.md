# ADR-0094: Stop the setup daemon before updating Windows files

Status: Accepted · Date: 2026-11-22

The Windows installer must be able to close Kongtrol before the first VPN exists
and after the user clears their data. In v0.8.0-rc.3, both shutdown attempts loaded
a VPN config before reaching the running process. Windows Restart Manager then
found the executable still in use.

Close the tray before requesting daemon shutdown. The new binary's hidden
`down --all --remember` path reads the listener settings and saves the connected
profiles through the local API. The daemon's own shutdown handles connections,
routes and DNS. Setup records failures running the bundled command and stops at
the preparation page with recovery steps. Restart Manager remains available for
files still in use after preparation.

The Windows release job exercises a reinstall while Kongtrol serves the setup
dashboard without any VPN profile. This refines the preparation step in
[ADR-0050](0050-installer-resume-tunnels-and-open-request.md).

[Research](../research/2026-11-22-windows-installer-shutdown.md)
