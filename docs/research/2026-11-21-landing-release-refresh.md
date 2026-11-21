# Research: Landing structure, motion and release freshness

Version 1.0.0 · 2026-11-21

| Source | Accessed |
|---|---|
| [WireGuard project overview](https://www.wireguard.com/) and [Quick Start](https://www.wireguard.com/quickstart/) | 2026-11-21 |
| [Cloudflare Workers Builds deploy hooks](https://developers.cloudflare.com/workers/ci-cd/builds/deploy-hooks/) | 2026-11-21 |
| [GitHub REST API: latest release](https://docs.github.com/en/rest/releases/releases#get-the-latest-release) | 2026-11-21 |

The home page links to the instructions a new user needs. Advanced compatibility
and FAQ content can be opened when needed. The routing demo was paused on hover
and stopped permanently after a manual selection. The download curve was drawn
by a scroll animation, leaving the lines partly drawn in the captured position.

The existing landing reads the latest stable release during the build. It could
reuse a cached tag, and publishing a tag does not itself trigger its main-branch
site build. GitHub listed `v0.7.0` as the latest stable release when checked.

Configure a Workers Builds deploy hook for the production branch and store the
URL as the repository Actions secret `CLOUDFLARE_LANDING_DEPLOY_HOOK`. The stable
release job calls it only after GoReleaser publishes. Verify the resulting build
in Cloudflare's build history. The live production hook is **no verificado**.
