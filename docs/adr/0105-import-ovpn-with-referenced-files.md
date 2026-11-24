<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0105: Import an .ovpn with the files it names: native file picker, several files at once, "Agregar <name>"

Version 1.0.0 · 2026-11-24

Status: Accepted · Date: 2026-11-24 · Decision-makers: Ruben Luna (owner, option A), implemented by agent

Refines [ADR-0042](0042-dashboard-api-completeness.md) (an imported `.ovpn` had to be
self-contained). Research: [docs/research/2026-11-24-native-file-picker.md](../research/2026-11-24-native-file-picker.md).

## Context and Problem Statement

Many providers ship an `.ovpn` next to `ca.crt`, `ta.key` or a client certificate and name them
(`ca ca.crt`, `tls-auth ta.key 1`). The panel's "Importar un archivo de VPN" refused them with
`api.err.import_file_ref` even when the files were in the same folder: the browser gives the page
a file name and its content, never a path, and `profileimport.Validate` passed the helper's
sanitizer a reader that refused every reference. The refusal told a novice to edit `config:` by
hand. `kongtrol profile import` refused them the same way.

## Decision Drivers

- The easiest path for a non-technical user; full parity for technical users (CLI).
- Reuse what exists: the helper's sanitizer already inlines referenced files.
- The stored profile stays self-contained (the helper never opens another user path).
- Security: no path traversal, no links, size caps, read with the user's own rights, nothing of
  the files in logs or the audit log.
- Prefer a vetted library over our own OS dialog code.

## Considered Options

1. Keep refusing; better wording only.
2. **A (chosen):** a native file picker through the daemon (real path → references read from the
   file's folder), plus several files at once in the browser (drop or input) and an
   "Agregar <name>" button for what is still missing.
3. Tell users to point a profile at the original file (no import), as the old hint said.

## Decision Outcome

Chosen: option A (owner-approved).

- `internal/profileimport/refs.go`: `Refs{Dir, Files}` and `Resolve` run
  `privops.SanitizeOpenVPNConfig` with a reader. A reference is matched first against files added
  by hand (base name, then case-insensitive), then read from `Dir` (the picked file's folder,
  like OpenVPN GUI's `--cd`): only inside that folder or its subfolders (absolute paths and `..`
  that leave it are never opened, they count as missing; `C:x` and `:` streams on Windows too);
  every folder on the way a real folder (no link, no junction); the file a regular file, not a
  link, opened without following one (`O_NOFOLLOW` on Unix), checked again after opening
  (`os.SameFile`), at most `privops.MaxReferenceSize` (256 KiB). All missing files are collected
  first and returned together (`MissingFilesError` → `api.err.import_missing_files`). Which
  directives may name a file stays the helper's list (`ca`, `cert`, `key`, `tls-auth`,
  `tls-crypt`, `tls-crypt-v2`, `secret`, `dh`, `extra-certs`, `crl-verify`, `pkcs12`);
  `auth-user-pass <file>` and the denylist stay refused. The stored copy is the sanitized,
  self-contained text (≤ 512 KiB), only when something was inlined; otherwise the original bytes.
  Added files: at most 8, each ≤ 256 KiB, all together ≤ 512 KiB.
- New keys (ES+EN, each with a "what to do" line): `api.err.import_missing_files`,
  `import_ref_unreadable`, `import_ref_too_big`, `import_ref_not_text`, `import_too_many_files`,
  `picker_unavailable`, `picker_busy`. `api.err.import_file_ref` remains only where no reader
  exists (the elevated root-only read, ADR-0058) and no longer tells a novice to edit `config:`;
  editing the file stays as a secondary hint.
- API: `POST /api/v1/vpns/import` takes `files: [{name, content_base64}]`; a `path` import reads
  references from the file's folder (not for the elevated read); the answer adds `included`; a
  400 `api.err.import_missing_files` adds `missing: [names]`. The body cap for this route is
  2 MiB (1 MiB elsewhere). `GET /api/v1/vpns/import/pick` → `{available}`; `POST
  /api/v1/vpns/import/pick` opens the OS dialog (`internal/filepicker`, `github.com/ncruces/zenity`
  v0.10.15, MIT, no cgo) and answers `{path, name, type, content_base64}` or `{cancelled: true}`;
  503 `picker_unavailable`, 409 `picker_busy` (one dialog at a time), 10 min bound, no config lock.
  The picked file goes through the same checks as a `path` import. Audit action
  `config.profile.import_pick` (who / what / result; never the path or content).
- Panel (`web/ui` Add sheet `FileStep`): "Elegir archivo" uses the OS dialog when available, else
  the browser's input (`multiple`); a drop takes every file (the first `.conf`/`.ovpn` is the VPN
  file, the rest its certificates; the window-wide drop passes them all). A missing-files refusal
  shows "Agregar <name>" per file; adding (button or drop) imports again with them. A stored copy
  belongs to one choice (`id`), so a newer file never reuses it, and an import answer for an older
  choice is dropped. The profile editor's "Importar archivo" sends dropped/picked extras too.
- Drop zone flicker: a shared depth counter (`useDropZone`), `dragenter` cancelled as the HTML
  drag-and-drop model expects, instead of "any dragleave turns it off".
- CLI: `kongtrol profile import <file> [certificates…]` reads references from the file's folder
  and matches the extra arguments by base name; a missing-files error adds the command to pass
  them.
- On Windows the daemon also raises the dialog in z-order once it appears (`SetWindowPos`
  topmost then not topmost, then `SetForegroundWindow`), because the documented foreground rules
  do not let a background process take focus while the user works in the browser; routing the
  dialog through the tray would not change that (it did not start the browser nor receive the
  click).

### Consequences

- Good: an `.ovpn` with sibling certificates imports in one click from the native dialog, or by
  dropping the files together; what is missing is named with a button to add it.
- Good: the stored profile and the helper's rules do not change; nothing new is privileged.
- Bad: a new dependency (zenity; its module graph also lists tool modules it uses for its own
  builds). KDE sessions without zenity and the XDG portal are not covered (browser fallback).
- Neutral: a network share (UNC path) picked in the dialog is refused like any UNC `path` import.

### Confirmation

Go tests: `internal/profileimport/refs_test.go` (folder, subfolder, added files, all missing
listed, outside folder never read, links/folders/big/binary refused, limits),
`internal/api/handlers_vpn_import_refs_test.go` (path import, `missing`, `files`, picker
endpoint states), `internal/filepicker` (unavailable, cancel, busy), CLI
`TestProfileImport_ReadsReferencedFiles`. Panel tests in `AddSheet.test.tsx` (multi-file drop,
"Add <name>", replace without reusing the stored copy, no flicker, OS dialog by path, fallback).

**No verificado — prueba manual del dueño:** whether the dialog comes to the front on Windows
when the user has just clicked in the browser (one bounded probe from a detached process came to
the front, in conditions that may not match a real click), and placement on macOS and Linux. Steps
in the research note, "Manual test (owner)".
