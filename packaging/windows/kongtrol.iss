; Kongtrol per-user Windows installer (Inno Setup 6) — version 1.0.0 · 2026-11-18
;
; Built by .github/workflows/release.yml (job "windows"); decision and sources
; in docs/adr/0025-desktop-release-channels-first-run.md. Installs for the current
; user only (no admin): kongtrol.exe (CLI + daemon) and kongtrol-tray.exe side by
; side in %LOCALAPPDATA%\Programs\Kongtrol, a Start menu entry, optional
; autostart at sign-in, optional user PATH entry, and optionally the network
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
AppPublisher=VPN Kongtrol
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
es.TaskPath=Agregar el comando kongtrol a la terminal (opcional)
en.TaskPath=Add the kongtrol command to the terminal (optional)
es.TaskHelper=Instalar el ayudante de red: Windows pedirá permiso una sola vez (recomendado)
en.TaskHelper=Install the network helper: Windows asks for permission once (recommended)
es.RunHelper=Instalando el ayudante de red · acepta el aviso de Windows…
en.RunHelper=Installing the network helper · accept the Windows prompt…
es.LaunchTray=Abrir Kongtrol y crear mi primera VPN
en.LaunchTray=Open Kongtrol and create my first VPN

[Tasks]
Name: "autostart"; Description: "{cm:TaskAutostart}"
Name: "helper"; Description: "{cm:TaskHelper}"
Name: "path"; Description: "{cm:TaskPath}"; Flags: unchecked

[Files]
Source: "{#SourceDir}\kongtrol.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\kongtrol-tray.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\Kongtrol"; Filename: "{app}\kongtrol-tray.exe"

[Registry]
; Autostart for this user (HKCU Run key).
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "Kongtrol"; ValueData: """{app}\kongtrol-tray.exe"""; Tasks: autostart; Flags: uninsdeletevalue
; User PATH (HKCU\Environment); removed again by CurUninstallStepChanged.
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Tasks: path; Check: NeedsAddPath(ExpandConstant('{app}'))

[Run]
; `service install` asks for elevation itself (one UAC prompt).
Filename: "{app}\kongtrol.exe"; Parameters: "service install --quiet"; StatusMsg: "{cm:RunHelper}"; Tasks: helper; Check: IsFirstInstall; Flags: runhidden waituntilterminated
Filename: "{app}\kongtrol-tray.exe"; Description: "{cm:LaunchTray}"; Flags: postinstall nowait skipifsilent
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
var
  WasInstalled: Boolean;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  Exe: string;
  Code: Integer;
begin
  Result := '';
  Exe := ExpandConstant('{app}\kongtrol.exe');
  WasInstalled := FileExists(Exe);
  if WasInstalled then
    Exec(Exe, 'down --all --quiet', '', SW_HIDE, ewWaitUntilTerminated, Code);
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/f /im kongtrol-tray.exe', '', SW_HIDE, ewWaitUntilTerminated, Code);
end;

// The helper is installed on the first install only: on upgrades the tray
// and the dashboard offer to reinstall it when its protocol version changed
// (privops.ProtocolVersion), so an upgrade never asks for UAC needlessly.
function IsFirstInstall: Boolean;
begin
  Result := not WasInstalled;
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

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Path: string;
begin
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
