# Research: WireGuard routing and TunnelBear diagnostics on Windows

Sources accessed 2026-11-21.

- [WireGuard Windows networking](https://git.zx2c4.com/wireguard-windows/tree/docs/netquirk.md?h=v1.1.1&id=be08995874ca5f6f3b74a0fb1d084e541d121c45) documents /1 pairs and Windows DNS behavior. Its native single-peer /0 firewall would block traffic through another VPN.
- [WireGuard Windows config parser](https://git.zx2c4.com/wireguard-windows/tree/conf/parser.go) accepts automatic routing and supports no SaveConfig option. Kongtrol omits the validated no-op value when preparing its private Windows copy.
- [Windows route table](https://learn.microsoft.com/en-us/windows/win32/api/netioapi/ns-netioapi-mib_ipforward_row2) and [Microsoft Get-NetAdapter](https://learn.microsoft.com/en-us/powershell/module/netadapter/get-netadapter?view=windowsserver2025-ps) document the properties used to identify candidate conflicting routes and distinguish physical adapters.
- [TunnelBear driver troubleshooting](https://help.tunnelbear.com/hc/en-us/articles/360060998872-Error-Driver-Error-on-Windows) identifies the maintenance process and TunnelBear Adapter v9 as possible causes of a driver error. Its [protocol guide](https://help.tunnelbear.com/hc/en-us/articles/360061409531-Error-Windows-Stop-Code) explains that IKEv2 can use system adapters. An absent named adapter cannot establish that TunnelBear is broken.
- [Windows PnP device state](https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-pnpentity) exposes ConfigManagerErrorCode. Diagnostics report the code for a matching TunnelBear network device, including a disabled device, without making any system changes.
