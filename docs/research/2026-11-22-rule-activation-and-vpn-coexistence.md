# Research: Rule activation, connection retries and Windows coexistence

Sources checked 2026-11-22.

- [AWS retry guidance](https://docs.aws.amazon.com/wellarchitected/2022-03-31/framework/rel_mitigate_interaction_failure_limit_retries.html)
  recommends explicit attempt limits and exponential spacing. Kongtrol keeps its
  previous three-minute spacing for the first retry and consumes budget only
  when it sees new activity. Jitter is unnecessary for a single local daemon.
- [WireGuard for Windows networking notes](https://git.zx2c4.com/wireguard-windows/about/docs/netquirk.md)
  describe Windows routes, DNS and the service's special firewall treatment
  for default AllowedIPs. A narrow routing policy alone cannot guarantee
  coexistence with another application's filtering.
- [Proton VPN for Windows source](https://github.com/ProtonVPN/win-app/blob/master/src/Configurations/ProtonVPN.Configurations/Defaults/DefaultConfiguration.cs)
  includes ProTUN configuration and a conflicting adapter checker.
  [Proton's kill switch documentation](https://protonvpn.com/support/advanced-kill-switch)
  also describes filters that block other traffic until Proton connects if its
  advanced kill switch is selected.
- [Inno Setup task parameters](https://jrsoftware.org/ishelp/topic_componentstasksparams.htm)
  and [command line options](https://jrsoftware.org/ishelp/topic_setupcmdline.htm)
  describe task-gating and explicit selection in interactive and quiet setups.

In the reported Windows case, the browser saw a network change during an
automatic attempt. Proton named the Claudio WireGuard adapter as a possible
conflict and reconnected after Kongtrol exited. A live local adapter check
found Proton's ProTUN interface after recovery. The exact packet filter that
interrupted the browser was not verified.
