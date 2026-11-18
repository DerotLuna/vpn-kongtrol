<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: VPN provider logos on the landing (brand guidelines)

Version 1.1.0 · 2026-11-18

Outcome (1.1.0): the maintainer chose our own monograms for all ten VPNs, Proton included
(ADR-0023). The verdicts below record what each brand allows; none of their logos is used.

## Question

Can the landing's VPN section show each supported VPN's official logo — monochrome, from
[Simple Icons](https://simpleicons.org) 16.38.0 (CC0), self-hosted and drawn in `currentColor` —
next to its name, for an independent open-source project with no affiliation? For ADR-0023.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Simple Icons — package 16.38.0 (`icons/*.svg`, `data/simple-icons.json`) | https://cdn.jsdelivr.net/npm/simple-icons@16.38.0/ | 2026-11-17 |
| 2 | Simple Icons — legal disclaimer (CC0 covers the files, not the trademarks) | https://github.com/simple-icons/simple-icons/blob/develop/DISCLAIMER.md | 2026-11-17 |
| 3 | NordVPN — press area | https://nordvpn.com/press-area/ | 2026-11-17 |
| 4 | Nord Security — trademark and brand guidelines | https://nordvpn.com/trademark-policy/ | 2026-11-17 |
| 5 | Proton — media kit | https://proton.me/media/kit | 2026-11-17 |
| 6 | Surfshark — press assets | https://surfshark.com/press/assets | 2026-11-17 |
| 7 | Fortinet — legal (section 5, Trademarks) | https://www.fortinet.com/corporate/about-us/legal | 2026-11-17 |
| 8 | WireGuard — trademark policy | https://www.wireguard.com/trademark-policy/ | 2026-11-17 |
| 9 | OpenVPN — legal / trademark | https://openvpn.net/legal/ | 2026-11-17 |
| 10 | Tailscale — press page and Brand Toolkit (April 2025, in the media kit zip) | https://tailscale.com/press | 2026-11-17 |
| 11 | Cloudflare — trademark guidelines | https://www.cloudflare.com/trademark/ | 2026-11-17 |
| 12 | Cisco — corporate logo usage | https://www.cisco.com/c/en/us/about/brand-center/logo-usage-guidelines/corporate-logo.html | 2026-11-17 |
| 13 | Palo Alto Networks — brand page and Brand Guidelines July 2026 (PDF) | https://www.paloaltonetworks.com/company/brand | 2026-11-17 |
| 14 | Palo Alto Networks — terms of use | https://www.paloaltonetworks.com/legal-notices/terms-of-use | 2026-11-17 |

Quotes were captured with an automated fetch tool or a web search summary; re-open the URL before
quoting them elsewhere.

## Findings

Simple Icons "is released under CC0 - though that doesn't mean to imply that all icons within the
project are also CC0" and asks users to check each icon's licence and the brand's terms [2]. None
of the ten entries carries a `license` field in 16.38.0 [1]. So the deciding factor is each
brand's own policy. In 16.38.0 Simple Icons links guidelines for
NordVPN, WireGuard, OpenVPN, Cloudflare and Palo Alto Networks only [1].

| Brand (slug) | What the owner says | Verdict |
|---|---|---|
| NordVPN (`nordvpn`) | Marks "should be used exactly as provided"; "Don't change the Nord trademark colors"; other uses need a request to press@nordsec.com; descriptive (fair) use of the name is allowed [4]. Press area: use the original logo, the white version only on dark backgrounds [3]. Both pages answered HTTP 403 to the fetch tool; the wording comes from the search index: **no verificado** on the live page. | Monogram. A `currentColor` recolour changes the trademark colours. |
| Proton VPN (`protonvpn`) | "These images and brand assets may be freely used, provided the accompanying media refers back to proton.me." Nothing on colour or modification [5]. | Logo allowed with a link to proton.me. Not used: monogram for consistency (ADR-0023). |
| Surfshark (`surfshark`) | "For non-commercial use only"; "You may not reproduce, modify, create derivative works from or sell any Logos & Media Assets" [6]. | Monogram. A recolour is a modification. |
| FortiClient (`fortinet`) | "Users are not permitted to use these Marks without the prior written consent of Fortinet" [7]. | Monogram. |
| WireGuard (`wireguard`) | Section B.5 bars using the marks to suggest "compatibility" or "relatedness"; the logo must not be modified and its colour is fixed at `#88171a` [8]. | Monogram. |
| OpenVPN (`openvpn`) | The keyhole logo needs "prior written consent of OpenVPN"; the word mark needs the ® statement except when describing the Community software [9]. | Monogram. |
| Tailscale (`tailscale`) | No third-party rule published. The Brand Toolkit says "Avoid using the logo in single-color formats" and a 48×48 px minimum [10]. | Monogram. The Simple Icons version is single-colour. |
| Cloudflare WARP (`cloudflare`) | Logos need written permission (except Web Badges for customers); to show compatibility use the word mark with a referential phrase such as "compatible with" [11]. | Monogram. |
| Cisco AnyConnect (`cisco`) | Using the logo "to imply affiliation with or endorsement by Cisco" without "express written permission from Cisco is strictly prohibited"; exceptions only for news media, education, sponsored events [12]. | Monogram. |
| GlobalProtect (`paloaltonetworks`) | The brand page and guidelines cover only Palo Alto Networks' own use; co-branding needs Creative Services approval [13]. The terms grant no trademark licence [14]. | Monogram. |

Name text (not logos), noted for later:

- WireGuard B.5 also covers the word mark when it suggests compatibility [8]. Kongtrol runs
  WireGuard tunnels with the official tools, which section A.iii may cover; **no verificado**.
  Out of scope here: the name stays, as in the rest of the site and the docs.
- OpenVPN asks for "OpenVPN® is a registered trademark of OpenVPN Inc." outside descriptive use [9].
  The section's note ("names are trademarks of their respective owners") is a general
  attribution; whether it is enough is **no verificado**.

## Confidence

Medium-high. Primary sources for every brand except NordVPN (blocked by HTTP 403, read through
the search index). This is not legal advice: the conservative reading wins where a policy is
silent or unclear.

## Open questions

- Written permission from any brand would let it show a logo; asking is the maintainer's call.
- Re-check the policies when Simple Icons or the brands' pages change.

## Used by

- ADR-0023
