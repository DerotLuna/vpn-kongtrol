# Rule suggestions and notices

Accessed 2026-11-26.

- [OpenAI network recommendations](https://help.openai.com/en/articles/9247338-network-recommendations-for-chatgpt-errors-on-web-and-apps) lists ChatGPT and OpenAI domains, including the static and upload domains, and notes that WebSockets and voice can use additional destinations.
- [Anthropic tenant restrictions](https://support.claude.com/en/articles/13198485-enforce-network-level-access-control-with-tenant-restrictions) identifies `claude.ai`, `claude.com`, `anthropic.com` and `api.anthropic.com`. It is not a complete list of Claude's dependencies.
- [Chrome toast guidance](https://developer.chrome.com/docs/modern-web-guidance/guides/ui-components/persistent-toast-notifications) recommends persistent important notices and another way to retrieve transient ones.
- [Red Hat alert accessibility](https://ux.redhat.com/patterns/alert/accessibility/) recommends persistent important messages and non-timed alternatives for actions.

An allowlist is not a VPN routing guarantee. Kongtrol matches `*.` against one
label and probes only a small set of common subdomains for routes
(`internal/policy/rules.go`, `internal/monitor/policyresolver.go`). Shared
third-party services and voice IP ranges are not silently included. The user
must review destinations and can add or remove any of them.
