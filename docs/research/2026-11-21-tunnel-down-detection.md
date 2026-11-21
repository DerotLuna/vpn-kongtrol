# Observing TCP attempts while policy traffic is blocked

Accessed 2026-11-21. Used by [ADR-0084](../adr/0084-per-profile-tunnel-down-protection.md).

- Windows `GetExtendedTcpTable` enumerates TCP endpoints available to the caller. Windows Filtering Platform offers separate blocked-connection audit events, including event 5157. Correlation to Kongtrol's helper rules on supported Windows versions is **unverified**. [Microsoft TCP table reference](https://learn.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getextendedtcptable), [Microsoft WFP audit reference](https://learn.microsoft.com/en-us/windows/win32/fwp/auditing-and-logging)
- Linux `/proc/net/tcp` and `/proc/net/tcp6` expose current TCP sockets. Netfilter can expose firewall rule logs through NFLOG. Observing only the helper's rejects through that mechanism would need privileged changes and system verification. [Linux kernel TCP table reference](https://docs.kernel.org/networking/proc_net_tcp.html), [Netfilter NFLOG documentation](https://netfilter.org/projects/libnetfilter_log/index.html)
- The existing Kongtrol TCP observer has implementations for Windows and Linux. OpenBSD documents PF rule logging, but its availability and permissions on supported macOS releases are **unverified**. [OpenBSD PF reference](https://man.openbsd.org/pf.conf)

The samples provide best-effort notices and connection triggers. The leased destination rules operate independently of sampling. Native observation of a TCP attempt after its firewall rejection remains **unverified**.
