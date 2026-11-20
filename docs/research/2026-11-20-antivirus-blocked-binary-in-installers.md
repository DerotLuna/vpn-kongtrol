<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Installers and a binary the antivirus blocks

Version 1.0.0 · 2026-11-20 · Backs [ADR-0073](../adr/0073-installers-stop-when-the-new-binary-cannot-run.md)

All sources accessed 2026-11-20.

## What happened (owner report, Windows 11, PowerShell 7.6)

`iwr https://kongtrol.app/install.ps1 -useb | iex` over v0.7.0-rc.2 installed v0.6.0 (latest
stable) and printed "Updating Kongtrol  -> latest" and "Kongtrol updated to latest". Microsoft
Defender had flagged the downloaded, checksum-verified `kongtrol.exe` as
`Trojan:Win32/Bearfoos.A!ml`, refused to start it ("Operation did not complete successfully
because the file contains a virus or potentially unwanted software") and quarantined it.
`Get-KongtrolVersion` swallowed the error and returned nothing for both the old and the new exe,
so the downgrade warning (ADR-0046) did not fire and the script claimed success.

## Windows: how a blocked start looks to a script

| Fact | Source |
|---|---|
| `ERROR_VIRUS_INFECTED` = 225: "Operation did not complete successfully because the file contains a virus or potentially unwanted software." (the owner's exact message) | https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes--0-499- |
| `ERROR_VIRUS_DELETED` = 226: "...the file has been removed from this location." | same |
| PowerShell 7.6 and Windows PowerShell 5.1: `& $exe` that cannot start throws `ApplicationFailedException`; inside a pipeline (`& $exe version 2>$null \| Select-Object -First 1`) its inner exception is a `System.ComponentModel.Win32Exception` with `NativeErrorCode`. | Measured here with a non-program `.exe` (code 216 / 193); 225 itself **no verificado** (we did not trigger a real detection on this machine) |
| Outside a pipeline PowerShell 7.6 gave an `InvalidOperationException` inner exception instead, so the script keeps the pipeline form and also treats "file gone after the attempt" (quarantine) as blocked. | Measured here |

## Microsoft Defender: restore, allow, report

| Fact | Source |
|---|---|
| Protection history cards: "Threat found - action needed" offers **Quarantine** or **Allow on device**; "Threat quarantined" offers **Remove** or **Restore**; Restore puts the file back, Defender detects it again and **Allow on device** then lets it run. Microsoft advises Quarantine when unsure. | https://support.microsoft.com/windows/protection-history-f1e5fd95-09b4-46d1-b8c7-1059a1e09708 |
| False positives: submit the file to the Microsoft Security Intelligence portal, https://www.microsoft.com/wdsi/filesubmission/; "Every exclusion lowers your level of protection"; exclusions only sparingly. | https://learn.microsoft.com/en-us/defender-endpoint/defender-endpoint-false-positives-negatives |
| Path in the Windows Security app: Virus & threat protection > Protection history (restore steps). | https://learn.microsoft.com/en-us/defender-endpoint/restore-quarantined-files-microsoft-defender-antivirus |

Whether **Allow on device** also covers a fresh download of the same file on the next run of the
script is **no verificado**. The Spanish UI names used in the Inno message and the landing
("Seguridad de Windows > Protección antivirus y contra amenazas > Historial de protección",
"Permitir en el dispositivo", "Restaurar") are **no verificado** against a Spanish Windows.

## What `!ml` means

Microsoft's naming page (CARO scheme) only says: "A suffix that begins with **!** is an indicator
used by Microsoft internally" (https://learn.microsoft.com/en-us/defender-xdr/malware-naming).
Microsoft security blog posts use `!ml` on machine-learning detections (for example
`Behavior:Win32/...!ml`), and community answers read it as "machine learning". There is no
official definition: **no verificado**. Kongtrol's docs say "usually a machine-learning
detection", not more.

## macOS / Linux

| Fact | Source |
|---|---|
| A shell gives exit status 126 when a command is found but cannot be executed (permission denied, `noexec` mount), 127 when it is not found, and 128+N when killed by signal N. | POSIX Shell Command Language, 2.8.2 Exit Status for Commands, https://pubs.opengroup.org/onlinepubs/9799919799/utilities/V3_chap02.html |
| macOS kills a process whose code signature it rejects with SIGKILL (shell: "Killed: 9", status 137). | Reasoned from common reports; **no verificado** in Apple documentation |

## How others handle it

Other projects' install scripts were not reviewed for this case in this session
(**no verificado**); running the downloaded binary once before installing it is a common
pattern, reasoned, not sourced. Our design: tell "blocked" apart from "no version", keep the old copy until the new one runs,
and record the installed version in the manifest for when the old exe cannot answer.

## Confidence

High for the Windows error codes and Defender actions (primary Microsoft docs) and for the
PowerShell exception shape (measured). Medium for the shell exit codes on macOS (POSIX plus
reasoning). Low for the meaning of `!ml` (no official source).

## Open questions

- Does "Allow on device" survive a fresh download (same hash) on the next run? Check on the next
  false positive.
- Code signing (SignPath Foundation, pending) should remove most of these detections.

## Used by

- ADR-0073
