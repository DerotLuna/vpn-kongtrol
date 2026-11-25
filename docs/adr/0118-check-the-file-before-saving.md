<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0118: The import checks the file as soon as it is chosen, before saving

Version 1.0.0 · 2026-11-25

Status: Accepted · Date: 2026-11-25 · Decision-makers: Ruben Luna (owner, approved design), implemented by agent

Refines [ADR-0042](0042-dashboard-api-completeness.md) (import endpoint),
[ADR-0105](0105-import-ovpn-with-referenced-files.md) (referenced files),
[ADR-0112](0112-find-a-dropped-ovpns-files-automatically.md) (dropped files, folder pick) and
[ADR-0115](0115-save-without-connecting.md) (the two save buttons). Research: none new (no external
mechanism adopted; the accessibility attributes are WAI-ARIA's own, see Confirmation).

## Context and Problem Statement

In the "＋ Agregar" file step the panel only learned what was wrong with a `.conf`/`.ovpn` (a hook
line, missing certificates, a file only root can read, a name already taken) after the user pressed
"Guardar" or "Guardar y activar". Saving was the first moment the daemon read the file, so the user
filled in the name and the password before finding out the file could not be used, and every
certificate added or folder picked meant pressing a save button again to learn whether it was now
complete.

## Decision Drivers

- Guide a lot, in short lines: say what is wrong (and what to do) as soon as the file is chosen.
- One set of rules: the answer must be exactly what the import would answer (ADR-0042 shares
  `internal/profileimport` between the API and the CLI); no second validator in the panel.
- Read nothing more than an import already reads; store nothing until the user saves.
- No added waiting: one request per user action, no polling.
- GUI/CLI parity.

## Considered Options

1. Keep checking only on save.
2. Validate in the panel (TypeScript copies of the helper's rules): two implementations that drift,
   and the panel cannot read the certificates next to the original or run the Locator.
3. **Chosen:** the same import with `"check": true`: the daemon runs every step and writes nothing.
4. A separate `POST /api/v1/vpns/import/check` route: the same body and answers on two routes, and one
   more route to audit; the flag keeps one route and one handler.

## Decision Outcome

Chosen: option 3.

- `profileimport.Request.Check` (`internal/profileimport/profileimport.go`): `Import` runs the name
  and type checks, the size cap, the validators, `Resolve` (folder, `ref_dir`, added files, outside
  references) and the `Locator` for a dropped file exactly as before, then stops before
  `os.MkdirAll`/`store`. Its `Result` carries `Check`, `File` (`<name>.conf|.ovpn`), `Exists`,
  `Included`, `FoundIn`, `Provider` and `NeedsCredentials` (`auth-user-pass`, also filled by a real
  import). There is no second validation path: the check is the import minus the write.
- **Name already taken:** a field, not a blocker. `Exists: true` when a regular file of that name is
  already in `~/.kongtrol/configs/`: the import would answer 409 `api.err.import_exists` and the user
  can still replace it or change the name, so the check passes and the panel warns early ("Ya hay un
  archivo guardado como …"). A folder or link in the way stays the refusal
  `api.err.import_not_file` (no import could store it).
- **API:** `POST /api/v1/vpns/import` with `"check": true` → 200
  `{"status":"checked","type","file","exists","included":[…],"missing":[],"needs_credentials",
  "found_in"?,"provider"?,"hint"?,"hint_key"?}`; every refusal keeps its status and body
  (`error`, `hint`, `error_key`, `missing`). A check never opens the OS permission window: `elevate`
  is ignored, so a root-only file answers 403 `config.err.file_root_only` and the user decides when
  saving. It takes no config lock (it writes nothing).
- **Audit (ADR-0064):** the middleware still never reads the body. The handler marks the request
  (`markRequest`, a request-scoped note created by `serveAnnounced`): it is audited as
  `config.profile.import_check` at INFO (who / what / result), and it does not bump the change hub,
  so no `changed` frame makes every client refetch. A real import stays `config.profile.import`
  (SECURITY).
- **Panel** (`web/ui/src/components/add/FileStep.tsx`, `useCheckProfileFile` in
  `src/data/profiles.ts`): every file chosen, dropped or picked, every certificate added and every
  folder picked runs the check. While it runs the result area shows a spinner and "Revisando el
  archivo…" (`aria-live="polite"`, `aria-busy`), in the same place the answer then takes (a reserved
  line, no layout jump). The answer appears at once: "Listo: OpenVPN…", the files that go inside,
  where they were found, a name already taken, the Proton hint, the username and password fields
  when needed; or the missing-files banner ("Elegir la carpeta donde están", "Agregar <name>"); or
  the refusal with its "what to do" line. "Guardar" and "Guardar y activar" stay off until the
  latest check passed, with the reason under them. A newer check aborts the previous one
  (`AbortController`) and leaving the step aborts it; an older answer is dropped. Adding a
  certificate or picking a folder now checks again instead of saving. Saving still runs the real
  import, and the daemon checks everything again.
- **CLI:** `kongtrol profile import <file> [certificates…] --check` runs the same local import with
  `Check` (no daemon needed, like the import), prints the evaluation translated (ready line, files
  inside, credentials, a name already taken with `--overwrite`/`--name`, Proton hint, next step) or
  the same refusal with its "what to do" line, and supports `--json` (the `Result`). It never asks
  for the root-only permission prompt. `--check` wins over `--create` and `--overwrite`.

### Performance

The check is one request per user action and no polling. It costs what the import already costs:
the validators run in memory on at most 512 KiB, the referenced files are a few small PEM files
(≤ 256 KiB each, ≤ 8 added), and the Locator for a dropped file is bounded by ADR-0112 (depth ≤ 3,
20 000 entries, 1 s). The save that follows repeats the work once; that is the price of the daemon
never trusting an earlier answer.

### Security

A check can read exactly what an import of the same body reads, with the same restrictions: an
absolute clean local path to a regular file (no link, junction, device or network path), the size
caps, the referenced files inside the folder or added by hand, references outside it only as PEM key
material (`api.err.import_ref_not_key`), and never through the elevated helper or the permission
prompt. It stores nothing (no folder, no file, no profile), its answer holds no file content (names,
type, the folder where certificates were found), and its audit entry has no body.

### Consequences

- Good: problems show while the user is still on the file, with what to do; saving only happens
  with a file that passed.
- Good: one implementation of the rules for the import, the check, the panel and the CLI.
- Neutral: a file is read twice (check, then save); bounded as above.
- Neutral: the panel checks with the name it proposed; if the user types another name, the save
  answer still covers a taken name (Keep / Replace).

### Confirmation

- Go: `internal/profileimport/check_test.go` (`TestImport_CheckWritesNothing`: no folder, no file, the
  same refusals, a taken name reported, a folder in the way refused; `TestNeedsCredentials`);
  `internal/api/handlers_vpn_import_check_test.go` (200 answer and fields, same refusal bodies, a
  check never elevates, audited as `config.profile.import_check` at INFO with no change announced);
  `cmd/kongtrol/cmd_profile_check_test.go` (`--check` saves nothing, same refusals, taken name).
- Panel: `web/ui/src/components/add/FileStep.test.tsx` "the file is checked as soon as it is chosen"
  (loader with `aria-busy`, both buttons off until the check passed, the passed answer, a refusal,
  a stale check aborted, closing the sheet aborts it) and the updated missing-files / folder tests;
  `AddSheet.test.tsx` adding a missing certificate checks again before saving.
- `aria-live` and `aria-busy` follow WAI-ARIA 1.2 (<https://www.w3.org/TR/wai-aria-1.2/#aria-live>,
  <https://www.w3.org/TR/wai-aria-1.2/#aria-busy>); not re-read for this change (**no verificado**
  beyond the existing panel use of the same attributes).
- **No verificado:** a manual run on Windows with a real OpenVPN GUI folder.
