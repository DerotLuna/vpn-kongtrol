<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0073: Installers stop when the new binary cannot run, and never claim an update that did not happen

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer

Refines [ADR-0046](0046-installers-detect-existing-install.md) (existing install, version table)
and [ADR-0050](0050-installer-resume-tunnels-and-open-request.md) (stop and resume on update).
Neither is superseded. Research:
[2026-11-20-antivirus-blocked-binary-in-installers](../research/2026-11-20-antivirus-blocked-binary-in-installers.md).

## Context and Problem Statement

On Windows 11, `install.ps1` over v0.7.0-rc.2 installed v0.6.0 and said "Updating Kongtrol
-> latest" and "Kongtrol updated to latest". Microsoft Defender had flagged the official,
checksum-verified `kongtrol.exe` (`Trojan:Win32/Bearfoos.A!ml`, a false positive on an unsigned
binary), refused to start it and quarantined it. The script asked each exe for its version, took
"could not start" as "no version", so: no older-version warning, empty versions in the messages,
the old copy replaced by a file that cannot run, and a success line at the end.

## Decision Drivers

- Never leave the user with a Kongtrol that cannot run, and never say "updated" when it is not.
- Say what happened and what to do in one or two lines (AGENTS.md "User guidance").
- The downgrade warning of ADR-0046 must still work when the old binary cannot answer.
- Security: never turn the antivirus off or add exclusions from a script; the user decides,
  after checking the file (Microsoft: "Every exclusion lowers your level of protection").
- Same behaviour in `install.ps1`, `install.sh` and the Windows installer.

## Considered Options

1. Probe the binary (`kongtrol version`) and tell "it ran" apart from "it could not start";
   stop before replacing, keep the old copy until the new one runs, record the version (chosen).
2. Keep going and only print a warning — rejected: the user ends with an unusable install.
3. Add a Defender exclusion or submit the file automatically — rejected: lowers protection,
   needs admin, and sends a file off the machine (AGENTS.md "Local only").

## Decision Outcome

Chosen option 1.

**`install.ps1` 1.7.0.** `Test-KongtrolExe` runs `<exe> version` and returns Ran (started, exit
0), Version, Blocked (Win32 error 225 `ERROR_VIRUS_INFECTED` / 226 `ERROR_VIRUS_DELETED`, or the
file gone right after the attempt) and the reason. The downloaded exe is probed before anything
changes: if it cannot run, `Write-CannotRun` prints what happened (blocked by an antivirus, or
Windows did not let it start, with Windows' own reason), that the SHA256 matched the release's
`checksums.txt`, and what to do (check the release with `gh attestation verify` or cosign; Windows
Security > Virus & threat protection > Protection history > Allow on device or Restore, only if
you trust it; run again; report at https://www.microsoft.com/wdsi/filesubmission), then stops
with "nothing was installed: your current Kongtrol <v> was not changed". After the move into place
and again before the success line the installed exe is probed; if it fails, the old copy
(`kongtrol.exe.old`) is put back and the script says so.

**`install.sh` 2.7.0.** Same flow with `bin_probe` / `cannot_run`: exit status 126 (not allowed to
run: `noexec`, security software), 127 (missing, quarantined), above 128 (killed by a signal; macOS
kills a rejected code signature with 9). A Linux 126 from the temp folder is retried once from
`~/.kongtrol` (a `noexec` `/tmp` is common on hardened systems and must not block installs that
worked before). The old binaries (Linux `kongtrol`, `kongtrol-tray` as `.old`; macOS
`Kongtrol.app.old`) stay until the new one runs and are put back otherwise. The final probe runs
before Kongtrol is opened.

**Previous version.** When the old binary cannot say its version, both scripts print "could not
read the installed version (<reason>)" and use a `version=` line that they now write into
`~/.kongtrol/install.manifest`, so 0.7.0-rc.2 -> 0.6.0 is still an older version. Without either,
the update line reads "Updating Kongtrol to <new>" (no empty "<old> ->"). `internal/uninstall`
ignores the new line (it acts only on absolute paths).

**Windows installer (`kongtrol.iss` 1.7.0).** Its version table reads the registry, not the exe,
so it is unaffected. But its `[Run]` entries assumed the copied `kongtrol.exe` runs and the
finished page said "Kongtrol se actualizó". New `BinaryRuns` (once, `Exec` of `kongtrol.exe
version`) gates every `[Run]` entry; when it fails an interactive install shows `BinaryBlocked`
(Windows' reason + what to do, ES/EN) and the finished page shows `FinishedBlocked` instead of
"updated". Inno has no `.old` copy to restore; the user re-runs the installer after allowing the
file. A Defender block while Inno copies the file is shown by Inno itself (its file error dialog).

The scripts stay English-only (ADR-0046); the Inno messages are ES/EN.

### Consequences

- Good: a blocked download leaves the installed Kongtrol untouched and says why and what to do;
  "updated" appears only after the installed binary ran.
- Good: the older-version warning works even when the old binary is blocked.
- Bad: one or two extra `kongtrol version` runs per install (milliseconds).
- Bad: the `.old` copies sit next to the new ones for the length of the install.
- Not verified: a real Defender detection (error 225) was not triggered here; the Windows
  exception shape was measured with a non-program `.exe` (error 216). The Inno script was
  reviewed, not compiled (no `iscc` here; CI compiles it on tags). shellcheck was not available
  here; CI's packaging job runs `sh -n` and the offline tests.

### Confirmation

- `bash packaging/tests/install-scripts.sh` (CI job `packaging`): new cases for a downloaded
  binary that cannot run (install.sh and install.ps1: stop, old copy untouched, no "updated")
  and for the manifest version fallback (0.3.0 -> 0.2.0 still an older version); the existing
  update case checks `version=` in the manifest and that no `.old` copy is left.
- PowerShell 7 and Windows PowerShell 5.1 parse `install.ps1`; `sh -n install.sh`.
- On the next release candidate, check by hand: Windows installer over an install whose new
  `kongtrol.exe` is quarantined (finished page and message).

## More Information

- User docs: landing guide (Windows install step), README "Antivirus or SmartScreen warning?",
  `docs/SECURITY.md` "Unsigned binaries and antivirus false positives".
