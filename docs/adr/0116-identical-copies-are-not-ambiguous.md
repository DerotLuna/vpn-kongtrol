<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0116: Identical complete copies of a dropped .ovpn are not ambiguous

Version 1.0.0 · 2026-11-25

Status: Accepted · Date: 2026-11-25 · Decision-makers: Ruben Luna (owner, real case), implemented by agent

Refines [ADR-0112](0112-find-a-dropped-ovpns-files-automatically.md). Research: none new (the
search bounds and sources are those of ADR-0112).

## Context and Problem Statement

ADR-0112 looks for the original of a dropped `.ovpn` and, when several folders hold an identical
copy, keeps the one holding every referenced file; still more than one is "ambiguous" and nothing is
guessed. The owner's own setup hit that rule: OpenVPN GUI keeps the same profile in
`%USERPROFILE%\OpenVPN\config\rluna.conf` and `%ProgramFiles%\OpenVPN\config\rluna\rluna.conf`, both
byte-identical, both folders with the same `ca.crt`, `rluna.crt`, `rluna.key` and `ta.key`. The drop
answered "missing files" although either folder gives exactly the same import.

## Decision Drivers

- Ask the user only what Kongtrol cannot find itself (ADR-0112).
- Never guess between copies that would import differently.
- Read no more than ADR-0112 already allows; keep every bound.
- Deterministic: the same disk gives the same answer every time.

## Considered Options

1. Keep refusing (ADR-0112 as is).
2. **Chosen:** when several folders are complete, compare every referenced file inside them (size
   and SHA-256); all identical → not ambiguous, use the first by root order; any difference → no
   guess, as before.
3. Prefer a fixed folder (e.g. always the user's OpenVPN folder) without comparing: imports the
   wrong certificates when the copies differ.

## Decision Outcome

Chosen: option 2.

- `Locator.Find` (`internal/profileimport/locate.go`) records the root index of each found folder.
  With two or more complete folders, `sameRefs` reads each relative reference in each of them like
  the copy itself is read (plain file, no link or reparse point, opened without following links and
  re-checked with `os.SameFile`, at most `MaxReferenceSize`) and compares SHA-256 (equal hashes
  imply equal sizes). An unreadable file counts as different. References outside the folder resolve
  to the same path from every folder and are not compared.
- All identical → the folder of the lowest root index wins; inside one root, the lexically smallest
  path (`File.ReadDir` order is not sorted). `SearchRoots()` puts `%USERPROFILE%\OpenVPN\config`
  before `%ProgramFiles%\OpenVPN\config`, so the owner's case uses the user's folder.
- Unchanged: depth ≤ 3, 20 000 entries, 1 s, 32 candidate hashes, no links, no network drives. The
  extra reads are only the referenced files of folders already chosen as complete (a few small PEM
  files each).

### Consequences

- Good: the owner's case (and any profile kept twice by OpenVPN GUI) imports in one step.
- Good: copies with any differing certificate or key still get no guess.
- Neutral: a few more small reads, only in the several-complete-copies case.

### Confirmation

Go: `internal/profileimport/locate_test.go` `TestLocate_SeveralCopies` (two complete identical
folders → the first by path; a different `ta.key` → none) and `TestLocate_IdenticalCopiesFirstRoot`
(the first root wins in either root order; a different `ta.key` → none).

**No verificado:** the owner's manual test on the real folders.
