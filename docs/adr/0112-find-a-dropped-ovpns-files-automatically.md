<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0112: Find a dropped .ovpn's files automatically, read absolute references, pick their folder

Version 1.0.0 · 2026-11-24

Status: Accepted · Date: 2026-11-24 · Decision-makers: Ruben Luna (owner: "automatízalo"), implemented by agent

Refines [ADR-0105](0105-import-ovpn-with-referenced-files.md). Research:
[docs/research/2026-11-24-find-ovpn-files-automatically.md](../research/2026-11-24-find-ovpn-files-automatically.md).

## Context and Problem Statement

ADR-0105 reads the files an `.ovpn` names from the folder of a file picked with a real path. A
browser drop has no path, so the owner's `rluna.conf`, dropped from `OpenVPN\config` with its four
certificates beside it, was refused with "missing files". Absolute references
(`ca C:\\certs\\ca.crt`) always counted as missing, even with a path. The missing-files banner
squeezed its text into one word per line in the narrow sheet, because its buttons sat beside it.

## Decision Drivers

- Ask the user only what Kongtrol cannot find itself (non-technical users).
- Never read more than needed: no content of unrelated files, no network shares, no links.
- Same rules for the panel and `kongtrol profile import`.
- Read with the user's own rights only; nothing new is privileged.

## Considered Options

1. Keep asking ("Agregar <name>" only).
2. **Chosen:** look for the dropped file's identical original in the usual folders; read absolute
   (and `../`) references with stricter checks; offer the OS folder dialog as the fallback.
3. Index the user's disk (Everything / Spotlight style): far more reading than needed.

## Decision Outcome

Chosen: option 2.

- **Locate** (`internal/profileimport/locate.go`, `Locator.Find`): when a dropped `.ovpn` (no
  folder) has missing files, the daemon searches `SearchRoots()` for a file with the same base name
  (`file_name` sent by the panel; else `<name>.ovpn|.conf`), then the same size, then the same
  SHA-256. Roots, most likely first: Windows `%USERPROFILE%\OpenVPN\config`,
  `%ProgramFiles%\OpenVPN\config`, Downloads, Desktop, Documents (Known Folder API, OneDrive
  redirection included); Linux XDG Downloads/Desktop/Documents, `~/.config/openvpn`,
  `/etc/openvpn/client`, `/etc/openvpn` (only if readable); macOS Downloads, Desktop, Documents,
  Tunnelblick `Configurations`, Viscosity `OpenVPN`. Bounds: depth ≤ 3, 20 000 entries, 1 s, 32
  hashes; links, junctions and reparse points are skipped (OneDrive cloud files accepted by tag,
  read without hydrating for the check); network drives and FUSE mounts skipped. One folder → used
  as the file's folder (the ADR-0105 checks apply) and the answer carries `found_in`; several →
  the one holding every relative reference (checked by name, not read); still several or none →
  no guess, missing files as before.
- **Absolute and `../` references** (`refs_disk.go` `readOutside`, `Refs.Disk`): read for the panel
  (path import, drop, folder pick) and the CLI, never for the elevated root-only read (ADR-0058).
  Checks: `CleanLocalPath` (no UNC, `\\?\`, `\??\`, device paths, `/dev`, `/proc`, `/sys`; on
  Windows drive letters only), not a network drive, every folder from the root a real folder, the
  file regular, ≤ 256 KiB, opened without following links and re-checked, and its content PEM
  blocks that an OpenVPN directive takes (`LooksLikeKeyFile`: CERTIFICATE, X509 CRL, PRIVATE KEY
  variants, DH PARAMETERS, OpenVPN static / tls-crypt-v2 keys). Otherwise
  `api.err.import_ref_not_key`. A non-local path counts as missing and is never opened. Why safe:
  the daemon runs as the user (OpenVPN GUI reads the same file with the same rights); of the
  inlined blocks only `cert` / `extra-certs` (or certificates inside `pkcs12`) reach the server,
  parsed as X.509 first; keys sign or key an HMAC and never leave; the PEM rule stops a hostile
  `.ovpn` from copying an OpenSSH key or a token file into `~/.kongtrol/configs`; a UNC path is
  never opened, so no NTLM handshake to another host. `pkcs12` outside the folder is refused (binary,
  not PEM): add it by hand.
- **Folder fallback**: `POST /api/v1/vpns/import/pick-folder` opens the OS dialog in folder mode
  (`filepicker.PickFolder`, zenity `Directory()`), answers `{path, name}` or `{cancelled: true}`
  (503 / 409 like the file dialog), only for a clean local real folder; the panel imports again
  with `ref_dir`, which replaces the file's folder. Audit `config.profile.import_pick_folder` (who /
  what / result, never the path). The browser-only fallback stays "Agregar <name>".
- **Panel**: the import sheet is 680 px wide (like the profile editor, still `100vw - 32px`); the
  missing-files banner reads "Faltan N archivos de este .ovpn", lists the names as chips, one line
  of what to do, and its buttons ("Elegir la carpeta donde están" — tonal, since the step's one gold
  button is "Guardar y activar" — then "Agregar <name>") wrap under the text, full width on a phone.
  A found original shows "Encontré los certificados junto a <file> en <folder>".
- **CLI**: `kongtrol profile import <file>` reads absolute and `../` references with the same checks.
- `CleanLocalPath` moved from `internal/api` to `internal/profileimport` (one implementation).

### Consequences

- Good: the owner's case imports in one step; a moved or copied `.ovpn` with absolute paths works.
- Good: nothing outside the candidate files (same name and size) and the chosen folder's referenced
  files is ever read.
- Bad: up to 1 s more before a missing-files answer for a drop with no original nearby.
- Neutral: a file outside the searched folders still needs the folder dialog or "Agregar".

### Confirmation

Go: `internal/profileimport/locate_test.go` (one match, other bytes, no name, several copies,
ambiguous, links not followed, entry/time/depth bounds, Import with `FoundIn`),
`refs_disk_test.go` (absolute and `../` refs, PEM-only outside, UNC/proc/links refused,
`LooksLikeKeyFile`, `ReferencedPaths`), `internal/api/handlers_vpn_import_locate_test.go` (drop finds
the original, `ref_dir`, absolute ref, pick-folder states), CLI `TestProfileImport_ReadsReferencedFiles`.
Panel: `web/ui/src/components/add/FileStep.test.tsx` (found message and `file_name`, chips and
stacked actions, folder button and `ref_dir`).

**No verificado:** the `IO_REPARSE_TAG_CLOUD` value `0x9000001A` and its variant mask (from
`winnt.h`, not checked this session; if wrong, OneDrive files are skipped, never followed); whether
OneDrive Files On-Demand originals are found in practice; the macOS and Linux client folders;
the owner's manual test (steps in the research note).
