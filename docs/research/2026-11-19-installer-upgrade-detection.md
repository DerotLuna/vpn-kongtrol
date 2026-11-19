<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Installers that detect an existing install

Version 1.0.0 · 2026-11-19 · Backs [ADR-0046](../adr/0046-installers-detect-existing-install.md)

All sources accessed 2026-11-19.

## Inno Setup: where the previous install is

| Fact | Source |
|---|---|
| "AppId also determines the actual name of the Uninstall registry key, to which Inno Setup tacks on `_is1` at the end." | https://jrsoftware.org/ishelp/topic_setup_appid.htm |
| `UsePreviousAppDir` (default yes): "Setup will look in the registry to see if the same application is already installed, and if so, it will use the directory of the previous installation". | https://jrsoftware.org/ishelp/topic_setup_usepreviousappdir.htm |
| `UsePreviousTasks` (default yes): reuses "the task settings of the previous installation as the default settings". | https://jrsoftware.org/ishelp/topic_setup_useprevioustasks.htm |
| `DisableDirPage=auto` (default): the folder page is not shown "if it finds the directory of a previous installation". | https://jrsoftware.org/ishelp/topic_setup_disabledirpage.htm |
| `UsePreviousLanguage` (default yes) pre-selects the previous language; `ShowLanguageDialog` default yes, `auto` shows it only without a language match. | https://jrsoftware.org/ishelp/topic_setup_usepreviouslanguage.htm, https://jrsoftware.org/ishelp/topic_setup_showlanguagedialog.htm |
| `DisableWelcomePage` default yes; the Ready page is never skipped by `DisableReadyPage` "if no other wizard page before the Ready to Install wizard page has been shown yet". | https://jrsoftware.org/ishelp/topic_setup_disablewelcomepage.htm, https://jrsoftware.org/ishelp/topic_setup_disablereadypage.htm |

The uninstall key lives under HKCU for a per-user (`PrivilegesRequired=lowest`) install and holds
`DisplayVersion` (= `AppVersion`) and `InstallLocation`: standard Windows uninstall values
(https://learn.microsoft.com/en-us/windows/win32/msi/uninstall-registry-key). That Inno writes
these two values for a per-user install is **no verificado** on a running system in this session;
the script falls back to "update, version unknown" when the version is missing.

## Inno Setup: event functions used

| Event | What the docs say | Source |
|---|---|---|
| `InitializeSetup` | "Called during Setup's initialization. Return False to abort Setup"; the wizard form does not exist yet. | https://jrsoftware.org/ishelp/topic_scriptevents.htm |
| `ShouldSkipPage` | Return True to skip the page; not called for `wpPreparing`, `wpInstalling`. | same |
| `CurPageChanged` | "Called after a new wizard page … is shown" (used to set titles, button caption, finished text). | same |
| `NextButtonClick` | Also "called on silent installs"; returning False before installation on a silent install makes Setup exit. So the downgrade question is asked only when `not WizardSilent`. | same, https://jrsoftware.org/ishelp/topic_isxfunc_wizardsilent.htm |
| `UpdateReadyMemo` | Returns the Ready memo text; the docs' default concatenation is rebuilt for a first install. | same |
| `MsgBox` with `MB_DEFBUTTON2` | Yes/No question defaulting to No (already used by the uninstaller). | https://jrsoftware.org/ishelp/topic_isxfunc_msgbox.htm |

Reasoned, **no verificado** (no `iscc` here): `RegKeyExists`, `RegQueryStringValue`, `CompareStr`,
`StrToIntDef`, `Trim`, `FmtMessage` and the `WizardForm.PageNameLabel` / `PageDescriptionLabel` /
`ReadyLabel` / `FinishedLabel` / `NextButton` members exist in Inno Setup 6 Pascal Script (all
listed in the Inno Setup help "Support Functions" and "Support Classes Reference"); the release
build compiles the script and fails on any mistake.

## Version order

SemVer 2.0.0 §11 precedence (https://semver.org/#spec-item-11): numeric `MAJOR.MINOR.PATCH`; a
pre-release version has lower precedence than the normal version; pre-release identifiers compare
left to right, numeric ones numerically, numeric below alphanumeric, alphanumeric in ASCII order,
and a larger set of fields wins when all preceding ones are equal; build metadata is ignored.
Release tags are `vX.Y.Z[-rc.N]` and `release.yml` passes `AppVersion` without the `v`.

## Precedent: how others present an update

- Inno-based apps (VS Code user installer, Git for Windows) reuse previous folder and choices on
  upgrade through the `UsePrevious*` defaults; Git for Windows also asks before a downgrade
  (seen in its `install.iss`, https://github.com/git-for-windows/build-extra/blob/main/installer/install.iss,
  **no verificado** in this session).
- Install scripts such as rustup (`rustup-init.sh`) report "updating existing installation"
  when one is found (https://github.com/rust-lang/rustup/blob/master/rustup-init.sh, **no
  verificado** in this session).
- Package managers (Homebrew, Scoop, winget, apt/dnf, npm, AUR helpers) print their own
  "Upgrading …" text, so Kongtrol adds nothing there.

## Finding in our code

`privops.Client.Call` returns `ErrHelperOutdated` when the reply's protocol differs
(`internal/privops/client.go`), so `Ping` against an older helper never returns a
`PingResult` with an older protocol. `outdatedHelperAction` treated that error as "not
installed", so `service install --if-outdated` (Windows upgrades) skipped an outdated helper.
Fixed in ADR-0046, with a test case.
