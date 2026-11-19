<img src="../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# How docs stay current

Version 1.1 · 2026-11-18

Rule: nothing may ever be out of date — enforced by tooling, not memory. Docs never block a merge:
code checks fail CI, doc checks only warn (annotations + one sticky PR comment).

## What is generated (never edit by hand)

| File | Source | Regenerate |
|---|---|---|
| `docs/CLI.md` (ES), `docs/CLI.en.md`, `docs/cli.json` | cobra command tree + `internal/i18n` (`help.*`, flag texts) | `make docs` |
| `docs/CONFIG.md` (ES), `docs/CONFIG.en.md` | `internal/config/schema.go` (fields, types, `validate` tags), `config.Defaults()`, descriptions in `internal/docgen/text.go` | `make docs` |
| Landing guide command reference | `docs/cli.json` (imported at build time by `landing/src/content/cmdref.ts`) | landing build |

`make docs` runs `go generate ./cmd/kongtrol` (hidden `kongtrol gen-docs`). Each generated file keeps its
version and date while unchanged; any change bumps the minor version and sets today's date.
`go run ./cmd/kongtrol gen-docs --check` (and `TestGeneratedDocsUpToDate`) report stale files as
warnings; set `KONGTROL_DOCS_STRICT=1` to make the tests fail locally.

Same approach as cobra's doc generator (`github.com/spf13/cobra/doc`, used by kubectl, gh and Hugo)
and Kubernetes' `hack/verify-generated-docs.sh`; ours renders one translated page per language plus
JSON, because `cobra/doc` only emits English headings.

## What is tested (blocking: code correctness)

- `internal/i18n`: ES/EN same keys, no empty text, same `fmt` arguments and verbs.
- `cmd/kongtrol`: every visible flag's text is an i18n key (so help and docs translate).
- `internal/config`: `configs/example.yaml` loads, validates and has no unknown key (strict decode).
- `web/ui`: `src/i18n/messages.test.ts`.
- `internal/impact`: every top-level package in `internal/` and `cmd/` is in `docs/impact.yml`.

Report-only: `landing` `pnpm run check:i18n` (empty texts, array lengths; keys are type-checked),
config fields without description, generated-doc freshness, links.

## Change impact map

`docs/impact.yml` maps each feature to its code globs and its surfaces (`path` or `path#anchor`).
Mark a paragraph with `<!-- feature:name -->` (Markdown) or `// feature:name` (TS/TSX/JS).
On every PR, `go run ./internal/impact/impactcheck` lists surfaces of touched features that did not
change, in the sticky comment: update them or write `no aplica: <motivo>` in the PR.

## Links

`lychee.toml` + `lycheeverse/lychee-action` over README, AGENTS.md, `docs/` and the built
`landing/dist`: internal links offline, external links with retries (403/429 accepted), weekly
scheduled run. localhost/127.0.0.1 are excluded. Locally: `lychee --offline README.md AGENTS.md docs`.

