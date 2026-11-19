; Kongtrol per-user Windows installer (Inno Setup 6) — version 1.3.0 · 2026-11-19
;
; Built by .github/workflows/release.yml (job "windows"); decision and sources
; in docs/adr/0025-desktop-release-channels-first-run.md; an existing install
; is detected and the wizard becomes "Actualizar" / "Reinstalar" / "versión
; anterior" (docs/adr/0046-installers-detect-existing-install.md). Installs for the current
; user only (no admin): kongtrol.exe (CLI + daemon) and kongtrol-tray.exe side by
; side in %LOCALAPPDATA%\Programs\Kongtrol, a Start menu entry, optional
; autostart at sign-in, the user PATH entry (on by default), and optionally the network
; helper (`kongtrol service install`, which shows one UAC prompt itself).
;
; Local build (Windows, Inno Setup 6 installed):
;   iscc /DAppVersion=1.2.3 /DSourceDir=..\..\build\win packaging\windows\kongtrol.iss

#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif
#ifndef SourceDir
  #define SourceDir "..\..\build\win"
#endif
#ifndef OutputDir
  #define OutputDir "..\..\build\installer"
#endif

[Setup]
; AppId identifies the product for upgrades and uninstall: never change it.
AppId={{BA277280-D8B2-4309-8620-BCEECBAA4B41}
AppName=Kongtrol
AppVersion={#AppVersion}
AppVerName=Kongtrol {#AppVersion}
AppPublisher=Kongtrol
AppPublisherURL=https://kongtrol.app
AppSupportURL=https://kongtrol.app/guia
DefaultDirName={autopf}\Kongtrol
DefaultGroupName=Kongtrol
DisableProgramGroupPage=yes
; Per-user install: {autopf} becomes %LOCALAPPDATA%\Programs, registry goes to
; HKCU. No UAC prompt for the install itself.
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir={#OutputDir}
OutputBaseFilename=Kongtrol-Setup-windows-amd64
SetupIconFile=..\..\assets\logo.ico
UninstallDisplayIcon={app}\kongtrol-tray.exe
UninstallDisplayName=Kongtrol
LicenseFile=..\..\LICENSE
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
; The user PATH changes: tell running programs (Explorer) to re-read it.
ChangesEnvironment=yes
; Upgrades: close a running tray through the Restart Manager.
CloseApplications=yes
RestartApplications=no

[Languages]
Name: "es"; MessagesFile: "compiler:Languages\Spanish.isl"
Name: "en"; MessagesFile: "compiler:Default.isl"

[CustomMessages]
es.TaskAutostart=Abrir Kongtrol al iniciar sesión (recomendado)
en.TaskAutostart=Open Kongtrol when you sign in (recommended)
es.TaskPath=Agregar el comando kongtrol a la terminal (recomendado)
en.TaskPath=Add the kongtrol command to the terminal (recommended)
es.TaskHelper=Instalar el ayudante de red: Windows pedirá permiso una sola vez (recomendado)
en.TaskHelper=Install the network helper: Windows asks for permission once (recommended)
es.RunHelper=Instalando el ayudante de red · acepta el aviso de Windows…
en.RunHelper=Installing the network helper · accept the Windows prompt…
es.RunHelperUpdate=Revisando el ayudante de red · si está desactualizado, acepta el aviso de Windows…
en.RunHelperUpdate=Checking the network helper · if it is out of date, accept the Windows prompt…
es.LaunchTray=Abrir Kongtrol y crear mi primera VPN
en.LaunchTray=Open Kongtrol and create my first VPN
es.LaunchTrayOpen=Abrir Kongtrol
en.LaunchTrayOpen=Open Kongtrol
; Existing install (docs/adr/0046-installers-detect-existing-install.md):
; %1 = installed version, %2 = this installer's version.
es.TitleUpdate=Actualizar Kongtrol %1 → %2
en.TitleUpdate=Update Kongtrol %1 → %2
es.TitleUpdateUnknown=Actualizar Kongtrol a %2
en.TitleUpdateUnknown=Update Kongtrol to %2
es.TitleRepair=Reinstalar Kongtrol %2 (reparar)
en.TitleRepair=Reinstall Kongtrol %2 (repair)
es.TitleDowngrade=Instalar una versión anterior de Kongtrol (%1 → %2)
en.TitleDowngrade=Install an older Kongtrol version (%1 → %2)
es.ReadyKeep=Tus VPN, ajustes y contraseñas se conservan. No hace falta elegir nada: se usan las opciones de la instalación anterior.
en.ReadyKeep=Your VPNs, settings and passwords are kept. Nothing to choose: the options of the previous install are reused.
es.ReadyDown=Las VPN conectadas se desconectan un momento. El ayudante de red solo se actualiza si hace falta (entonces Windows pide permiso una vez).
en.ReadyDown=Connected VPNs disconnect for a moment. The network helper is only updated if needed (then Windows asks for permission once).
es.BtnUpdate=Actualizar
en.BtnUpdate=Update
es.BtnRepair=Reinstalar
en.BtnRepair=Reinstall
es.BtnDowngrade=Instalar versión anterior
en.BtnDowngrade=Install older version
es.DowngradeAsk=Vas a instalar una versión anterior (%1 → %2).%n%nTus VPN y ajustes se conservan. Hazlo solo si la versión nueva te da problemas.%n%n¿Continuar?
en.DowngradeAsk=You are about to install an older version (%1 → %2).%n%nYour VPNs and settings are kept. Only do this if the new version gives you trouble.%n%nContinue?
es.FinishedUpdate=Kongtrol se actualizó a %2. Tus VPN y ajustes siguen ahí.%n%nSiguiente: abre Kongtrol y conecta tu VPN.
en.FinishedUpdate=Kongtrol was updated to %2. Your VPNs and settings are still there.%n%nNext: open Kongtrol and connect your VPN.
es.FinishedRepair=Kongtrol %2 se reinstaló. Tus VPN y ajustes siguen ahí.%n%nSiguiente: abre Kongtrol y conecta tu VPN.
en.FinishedRepair=Kongtrol %2 was reinstalled. Your VPNs and settings are still there.%n%nNext: open Kongtrol and connect your VPN.
es.PurgeAsk=¿Borrar también tu configuración y las contraseñas guardadas?%n%nSi piensas reinstalar, elige No.
en.PurgeAsk=Also delete your settings and saved passwords?%n%nIf you plan to reinstall, choose No.

[Tasks]
Name: "autostart"; Description: "{cm:TaskAutostart}"
Name: "helper"; Description: "{cm:TaskHelper}"
Name: "path"; Description: "{cm:TaskPath}"

[Files]
Source: "{#SourceDir}\kongtrol.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\kongtrol-tray.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\Kongtrol"; Filename: "{app}\kongtrol-tray.exe"

[Registry]
; Autostart for this user (HKCU Run key).
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "Kongtrol"; ValueData: """{app}\kongtrol-tray.exe"""; Tasks: autostart; Flags: uninsdeletevalue
; User PATH (HKCU\Environment); removed again by CurUninstallStepChanged.
; Prepended, so `kongtrol` in a new terminal is this copy rather than an
; older one earlier in the user PATH (the system PATH still comes first;
; `kongtrol doctor` warns about any other copy).
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{app};{olddata}"; Tasks: path; Check: NeedsAddPath(ExpandConstant('{app}'))

[Run]
; `service install` asks for elevation itself (one UAC prompt).
Filename: "{app}\kongtrol.exe"; Parameters: "service install --quiet"; StatusMsg: "{cm:RunHelper}"; Tasks: helper; Check: IsFirstInstall; Flags: runhidden waituntilterminated
; Upgrade (also silent): update a helper that speaks an older protocol, so
; it never stays outdated. A missing or current helper is left alone and
; no UAC prompt appears (`--if-outdated`, cmd/kongtrol/cmd_service.go).
Filename: "{app}\kongtrol.exe"; Parameters: "service install --if-outdated --quiet"; StatusMsg: "{cm:RunHelperUpdate}"; Check: IsUpgrade; Flags: runhidden waituntilterminated
Filename: "{app}\kongtrol-tray.exe"; Description: "{cm:LaunchTray}"; Check: IsFirstInstall; Flags: postinstall nowait skipifsilent
; Update / reinstall / older version: VPNs already exist, so only "open".
Filename: "{app}\kongtrol-tray.exe"; Description: "{cm:LaunchTrayOpen}"; Check: IsUpgrade; Flags: postinstall nowait skipifsilent
; Silent upgrade: bring the tray back (it was closed above).
Filename: "{app}\kongtrol-tray.exe"; Check: SilentUpgrade; Flags: nowait

[UninstallRun]
; Disconnect everything and stop the daemon, close the tray, remove the helper.
Filename: "{app}\kongtrol.exe"; Parameters: "down --all --quiet"; Flags: runhidden waituntilterminated; RunOnceId: "StopKongtrol"
Filename: "{sys}\taskkill.exe"; Parameters: "/f /im kongtrol-tray.exe"; Flags: runhidden waituntilterminated; RunOnceId: "StopTray"
Filename: "{app}\kongtrol.exe"; Parameters: "service uninstall --quiet"; Flags: runhidden waituntilterminated; RunOnceId: "RemoveHelper"

[Code]
// Upgrade over an existing install: kongtrol.exe (the daemon) is running and
// Windows locks its file. Stop it the normal way first (`down --all`: tunnels
// disconnect, DNS is restored and every kill switch block is lifted, so the
// user stays online), and close the tray. ~/.kongtrol, the keychain entries
// and the helper service are not touched.
//
// The previous install is detected at startup (InitializeSetup), before any
// page, from this AppId's uninstall key: Inno names it "<AppId>_is1" and,
// for this per-user install, writes it under HKCU with DisplayVersion =
// AppVersion and InstallLocation. Keep the GUID equal to AppId above.
// https://jrsoftware.org/ishelp/topic_setup_appid.htm
const
  UninstallKey = 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{BA277280-D8B2-4309-8620-BCEECBAA4B41}_is1';

  // InstallKind values.
  ikFirst = 0;      // nothing installed: the wizard stays as it was
  ikUpdate = 1;     // installed version < this one (or unknown)
  ikRepair = 2;     // same version
  ikDowngrade = 3;  // installed version > this one

var
  WasInstalled: Boolean;
  InstallKind: Integer;
  PrevVersion: string;

// ── Version compare (SemVer 2.0.0 §11 precedence, https://semver.org) ──
// Rules: an optional leading "v" is ignored and so is build metadata
// ("+..."); X.Y.Z compare numerically (a missing field counts as 0); a
// version with a pre-release ("-rc.3") is lower than the same version
// without one; pre-release identifiers compare dot by dot: numbers
// numerically, numbers below text, text in ASCII order, and a shorter list
// is lower when all its identifiers are equal. So
// 0.6.0-rc.3 < 0.6.0-rc.4 < 0.6.0 < 0.6.1 < 0.10.0.

// NextField returns S up to the first Sep and leaves the rest in S
// (all of S, and S empty, when there is no Sep).
function NextField(var S: string; Sep: string): string;
var
  P: Integer;
begin
  P := Pos(Sep, S);
  if P = 0 then
  begin
    Result := S;
    S := '';
  end else
  begin
    Result := Copy(S, 1, P - 1);
    S := Copy(S, P + Length(Sep), Length(S));
  end;
end;

function IsNumber(S: string): Boolean;
var
  I: Integer;
begin
  Result := Length(S) > 0;
  for I := 1 to Length(S) do
    if (S[I] < '0') or (S[I] > '9') then
    begin
      Result := False;
      exit;
    end;
end;

function CmpSign(N: Integer): Integer;
begin
  if N < 0 then
    Result := -1
  else if N > 0 then
    Result := 1
  else
    Result := 0;
end;

function CompareIdentifier(A, B: string): Integer;
begin
  if IsNumber(A) and IsNumber(B) then
    Result := CmpSign(StrToIntDef(A, 0) - StrToIntDef(B, 0))
  else if IsNumber(A) then
    Result := -1
  else if IsNumber(B) then
    Result := 1
  else
    Result := CmpSign(CompareStr(A, B));
end;

function StripV(S: string): string;
begin
  S := Trim(S);
  if (Length(S) > 0) and ((S[1] = 'v') or (S[1] = 'V')) then
    S := Copy(S, 2, Length(S));
  Result := S;
end;

// CompareVersions returns -1 when A < B, 0 when equal, 1 when A > B.
function CompareVersions(A, B: string): Integer;
var
  CoreA, CoreB, PreA, PreB, FieldA, FieldB: string;
begin
  A := StripV(A);
  B := StripV(B);
  PreA := NextField(A, '+');  // drop build metadata
  PreB := NextField(B, '+');
  CoreA := NextField(PreA, '-');  // PreA keeps the pre-release, if any
  CoreB := NextField(PreB, '-');
  repeat
    FieldA := NextField(CoreA, '.');
    FieldB := NextField(CoreB, '.');
    Result := CmpSign(StrToIntDef(FieldA, 0) - StrToIntDef(FieldB, 0));
    if Result <> 0 then
      exit;
  until (CoreA = '') and (CoreB = '');
  if (PreA = '') and (PreB = '') then
    exit;           // equal
  if PreA = '' then
  begin
    Result := 1;    // release > its pre-release
    exit;
  end;
  if PreB = '' then
  begin
    Result := -1;
    exit;
  end;
  repeat
    if (PreA = '') and (PreB = '') then
    begin
      Result := 0;
      exit;
    end;
    if PreA = '' then
    begin
      Result := -1; // fewer identifiers, all equal so far
      exit;
    end;
    if PreB = '' then
    begin
      Result := 1;
      exit;
    end;
    Result := CompareIdentifier(NextField(PreA, '.'), NextField(PreB, '.'));
  until Result <> 0;
end;

// ── Existing install ───────────────────────────────────────────────────
function InitializeSetup: Boolean;
var
  Dir: string;
begin
  Result := True;
  PrevVersion := '';
  Dir := '';
  WasInstalled := RegKeyExists(HKEY_CURRENT_USER, UninstallKey);
  if WasInstalled then
  begin
    RegQueryStringValue(HKEY_CURRENT_USER, UninstallKey, 'DisplayVersion', PrevVersion);
    RegQueryStringValue(HKEY_CURRENT_USER, UninstallKey, 'InstallLocation', Dir);
  end;
  if not WasInstalled then
    InstallKind := ikFirst
  else if PrevVersion = '' then
    InstallKind := ikUpdate     // installed, version unknown: treat as an update
  else
    case CompareVersions(PrevVersion, '{#AppVersion}') of
      -1: InstallKind := ikUpdate;
      0: InstallKind := ikRepair;
    else
      InstallKind := ikDowngrade;
    end;
  Log('Kongtrol: previous version "' + PrevVersion + '" in "' + Dir + '", kind ' + IntToStr(InstallKind));
end;

function KindText(Name: string): string;
begin
  Result := FmtMessage(CustomMessage(Name), [PrevVersion, '{#AppVersion}']);
end;

// Update, reinstall and older version need no choices: the license was
// accepted, the folder is the previous one (UsePreviousAppDir) and the
// task choices are the previous ones (UsePreviousTasks, both default yes),
// so the wizard goes straight to the Ready page.
// https://jrsoftware.org/ishelp/topic_setup_usepreviousappdir.htm
// https://jrsoftware.org/ishelp/topic_setup_useprevioustasks.htm
function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := (InstallKind <> ikFirst) and
    ((PageID = wpLicense) or (PageID = wpSelectDir) or (PageID = wpSelectTasks));
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  if InstallKind = ikFirst then
    exit;
  if CurPageID = wpReady then
  begin
    case InstallKind of
      ikRepair:
        begin
          WizardForm.PageNameLabel.Caption := KindText('TitleRepair');
          WizardForm.NextButton.Caption := CustomMessage('BtnRepair');
        end;
      ikDowngrade:
        begin
          WizardForm.PageNameLabel.Caption := KindText('TitleDowngrade');
          WizardForm.NextButton.Caption := CustomMessage('BtnDowngrade');
        end;
    else
      begin
        if PrevVersion = '' then
          WizardForm.PageNameLabel.Caption := KindText('TitleUpdateUnknown')
        else
          WizardForm.PageNameLabel.Caption := KindText('TitleUpdate');
        WizardForm.NextButton.Caption := CustomMessage('BtnUpdate');
      end;
    end;
    WizardForm.PageDescriptionLabel.Caption := CustomMessage('ReadyKeep');
    WizardForm.ReadyLabel.Caption := CustomMessage('ReadyDown');
  end
  else if CurPageID = wpFinished then
  begin
    // Older version: "Kongtrol <v> se reinstaló" reads right too.
    if InstallKind = ikUpdate then
      WizardForm.FinishedLabel.Caption := KindText('FinishedUpdate')
    else
      WizardForm.FinishedLabel.Caption := KindText('FinishedRepair');
  end;
end;

// The Ready page lists only the folder on an update: the previous task
// choices are kept and the helper task does not run again (see IsFirstInstall).
function UpdateReadyMemo(Space, NewLine, MemoUserInfoInfo, MemoDirInfo, MemoTypeInfo, MemoComponentsInfo, MemoGroupInfo, MemoTasksInfo: String): String;
begin
  if InstallKind <> ikFirst then
  begin
    Result := MemoDirInfo;
    exit;
  end;
  Result := '';
  if MemoUserInfoInfo <> '' then Result := Result + MemoUserInfoInfo + NewLine + NewLine;
  if MemoDirInfo <> '' then Result := Result + MemoDirInfo + NewLine + NewLine;
  if MemoTypeInfo <> '' then Result := Result + MemoTypeInfo + NewLine + NewLine;
  if MemoComponentsInfo <> '' then Result := Result + MemoComponentsInfo + NewLine + NewLine;
  if MemoGroupInfo <> '' then Result := Result + MemoGroupInfo + NewLine + NewLine;
  if MemoTasksInfo <> '' then Result := Result + MemoTasksInfo;
end;

// Older version: one confirmation, default No (MB_DEFBUTTON2). A silent
// install never asks (NextButtonClick also runs on silent installs).
// https://jrsoftware.org/ishelp/topic_scriptevents.htm
function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  if (CurPageID = wpReady) and (InstallKind = ikDowngrade) and not WizardSilent then
    Result := MsgBox(KindText('DowngradeAsk'), mbConfirmation, MB_YESNO or MB_DEFBUTTON2) = IDYES;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  Exe: string;
  Code: Integer;
begin
  Result := '';
  Exe := ExpandConstant('{app}\kongtrol.exe');
  // Files without an uninstall key (older or broken install) still count.
  WasInstalled := WasInstalled or FileExists(Exe);
  if FileExists(Exe) then
    Exec(Exe, 'down --all --quiet', '', SW_HIDE, ewWaitUntilTerminated, Code);
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/f /im kongtrol-tray.exe', '', SW_HIDE, ewWaitUntilTerminated, Code);
end;

// The helper task runs on the first install only. An upgrade runs
// `service install --if-outdated` instead: it asks for UAC only when the
// installed helper speaks an older protocol (privops.ProtocolVersion), so
// an upgrade never asks needlessly and never leaves the helper outdated.
function IsFirstInstall: Boolean;
begin
  Result := not WasInstalled;
end;

function IsUpgrade: Boolean;
begin
  Result := WasInstalled;
end;

function SilentUpgrade: Boolean;
begin
  Result := WasInstalled and WizardSilent;
end;

function NeedsAddPath(Dir: string): Boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Uppercase(Dir) + ';', ';' + Uppercase(OrigPath) + ';') = 0;
end;

// Uninstall keeps the user's data by default (docs/adr/0034-uninstall-and-update-notice.md). An
// interactive uninstall asks once, defaulting to No (MsgBox with
// MB_DEFBUTTON2); a silent one (UninstallSilent) never deletes data.
// https://jrsoftware.org/ishelp/topic_isxfunc_msgbox.htm
// https://jrsoftware.org/ishelp/topic_isxfunc_uninstallsilent.htm
var
  PurgeData: Boolean;

function InitializeUninstall: Boolean;
begin
  Result := True;
  PurgeData := False;
  if not UninstallSilent then
    PurgeData := MsgBox(CustomMessage('PurgeAsk'), mbConfirmation, MB_YESNO or MB_DEFBUTTON2) = IDYES;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Path: string;
  Code: Integer;
begin
  // Before any file is deleted: `uninstall --data-only` stops the tray and
  // the daemon itself (tunnels down, DNS restored, kill switch lifted), so
  // it does not depend on the [UninstallRun] order, then deletes
  // ~/.kongtrol and Kongtrol's keychain entries.
  if (CurUninstallStep = usUninstall) and PurgeData then
    Exec(ExpandConstant('{app}\kongtrol.exe'), 'uninstall --data-only --yes --quiet', '', SW_HIDE, ewWaitUntilTerminated, Code);
  if CurUninstallStep <> usPostUninstall then
    exit;
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path) then
    exit;
  Path := ';' + Path + ';';
  if StringChangeEx(Path, ';' + ExpandConstant('{app}') + ';', ';', True) > 0 then
  begin
    Path := Copy(Path, 2, Length(Path) - 2);
    RegWriteExpandStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path);
  end;
end;
