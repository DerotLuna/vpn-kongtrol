<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: public IP lookup services (ipify and alternatives)

Version 1.0.0 · 2026-11-17

## Question

Which external service should the opt-in public IP check call by default, and what do the
alternatives promise?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | ipify | https://www.ipify.org | 2026-11-17 |
| 2 | icanhazip | https://icanhazip.com | 2026-11-17 |
| 3 | Major Hayden — A new future for icanhazip | https://major.io/p/a-new-future-for-icanhazip | 2026-11-17 |
| 4 | ifconfig.co | https://ifconfig.co | 2026-11-17 |
| 5 | Cloudflare trace endpoint | https://1.1.1.1/cdn-cgi/trace | 2026-11-17 |
| 6 | SANS ISC diary 29516 (third party, lists checkip.amazonaws.com) | https://isc.sans.edu/diary/29516 | 2026-11-17 |
| 7 | Classmethod blog — checkip.amazonaws.com returns the X-Forwarded-For address (third party) | https://dev.classmethod.jp/articles/checkip-amazonaws-com-response-is-the-ip-address-of-x-forwarded-for-header-if-there-is-x-forwarded-for/ | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- ipify [1]: "ipify is completely open source"; "No visitor information is ever logged.
  Period."; "You can use it without limit". Endpoints: `api.ipify.org` (IPv4), `api6.ipify.org`
  (IPv6 only), `api64.ipify.org` (either). Plain-text answer.
- icanhazip [2][3]: the page returns only the IP. Its creator's post says Cloudflare continues
  "as the new owners and operators of icanhazip.com" (2021). Verified: Cloudflare-operated.
- checkip.amazonaws.com: **no verificado** as an officially documented AWS service — a
  docs.aws.amazon.com search found no page. Third-party evidence only: [6] lists it; [7] shows
  that behind proxies it returns the `X-Forwarded-For` address, so the answer may not be the
  real egress IP.
- ifconfig.co [4]: "the source code and documentation is available on GitHub"; "Please limit
  automated requests to 1 request per minute"; JSON with `Accept: application/json`. Repo name
  (`mpolden/echoip`) **no verificado** on the page.
- Cloudflare trace [5]: returns `key=value` lines including `ip`, `loc`, `colo`, `warp`. No
  documentation page found — undocumented endpoint, not a stable contract.

## Conclusions (reasoned)

- Default ipify: explicit no-logging statement, no rate limit, plain-text answer, open source.
- Allow user-set https endpoints instead of hard-coding several providers; never call more
  than one per lookup.
- Suggested endpoints (decision 2026-11-17): ipify (default) and icanhazip only.
  checkip.amazonaws.com was removed from the suggestions in `docs/SECURITY.md`,
  `docs/CONFIGURATION.md` and `configs/example.yaml` (no official AWS documentation; proxy
  header behaviour [7]). ifconfig.co (1 request/minute) and the Cloudflare trace (undocumented)
  are not suggested either.

## Confidence

Medium: the claims are the services' own statements, not audited.

## Open questions

- None.

## Used by

- [ADR-0010](../adr/0010-public-ip-check-opt-in.md)
