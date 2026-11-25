# Research: finding the files a dropped .ovpn names, and absolute references

Version 1.0.0 · 2026-11-24

## Question

ADR-0105 reads the files an `.ovpn` names from the picked file's folder, but a browser drop has no
path. The owner dropped `C:\Users\RLUNA\OpenVPN\config\rluna.conf` (refs `ca ca.crt`,
`key rluna.key`, `cert rluna.crt`, `tls-crypt ta.key`, all in that folder) and got "missing
files". He asked: look where the `.conf` is, or at the path the `.conf` names, before asking the
user. How do others solve it, how can the daemon find the original safely, and is reading an
absolute path an `.ovpn` names safe?

## Sources

| Source | Accessed |
|---|---|
| [OpenVPN 2.6 reference manual](https://openvpn.net/community-resources/reference-manual-for-openvpn-2-6/): `--cd`: "Change directory to dir prior to reading any files such as configuration files, key files, scripts, etc." | 2026-11-24 |
| [WHATWG HTML, file upload state](https://html.spec.whatwg.org/multipage/input.html#file-upload-state-(type=file)): the `value` prefixes the name with `C:\fakepath\`; "Some legacy user agents actually included the full path (which was a security vulnerability)" (read through a summary, not verbatim) | 2026-11-24 |
| [Microsoft: GetDriveTypeW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getdrivetypew): `DRIVE_REMOTE` "The drive is a remote (network) drive." | 2026-11-24 |
| [Microsoft: reparse point tags](https://learn.microsoft.com/en-us/windows/win32/fileio/reparse-point-tags): `IO_REPARSE_TAG_CLOUD` and `IO_REPARSE_TAG_CLOUD_1` … `_F` exist (no hex values on that page) | 2026-11-24 |
| [Microsoft: SHGetKnownFolderPath](https://learn.microsoft.com/en-us/windows/win32/api/shlobj_core/nf-shlobj_core-shgetknownfolderpath) (Known Folder API; `golang.org/x/sys/windows.KnownFolderPath`) | not fetched this session: **no verificado** |
| [freedesktop.org: xdg-user-dirs](https://www.freedesktop.org/wiki/Software/xdg-user-dirs/) (`user-dirs.dirs`, `XDG_DOWNLOAD_DIR="$HOME/…"`) | not fetched this session: **no verificado** |
| [Go 1.23 release notes, os](https://go.dev/doc/go1.23#os) and Go 1.27.4 `src/os/types_windows.go` (read locally): reparse points that are not links get `ModeIrregular`, except dedup files | 2026-11-24 (source read locally) |
| [RFC 8446 §4.4.2–4.4.3](https://www.rfc-editor.org/rfc/rfc8446#section-4.4.2) (TLS 1.3 Certificate / CertificateVerify): the client sends its certificate chain and a signature, never the private key | from knowledge, **no verificado** this session |
| [MITRE ATT&CK T1187 Forced Authentication](https://attack.mitre.org/techniques/T1187/) (a UNC path makes Windows authenticate to the remote host) | from knowledge, **no verificado** this session |
| [fdupes](https://github.com/adrianlopezroche/fdupes): README mentions signatures and a byte-for-byte confirmation, but not the order size → hash → bytes | 2026-11-24 |
| NetworkManager-openvpn `properties/import-export.c` | fetch failed (HTTP 406): **no verificado** |
| `privops.SanitizeOpenVPNConfig` (`internal/privops/ovpnconf.go`), read in the repo | 2026-11-24 |

## Findings

1. **Nobody else gets this case.** Desktop clients (OpenVPN GUI, Tunnelblick, Viscosity,
   NetworkManager) import from a native dialog, so they always have the path and resolve
   references against the config's folder (OpenVPN's own `--cd` rule). A browser-based panel is
   the odd one: the HTML spec hides the path on purpose (`C:\fakepath\`). No tool was found that
   searches for a dropped file's original; this is reasoned, not copied.
2. **Matching the original safely.** Duplicate finders compare size first and a hash only for
   same-size files (fdupes documents signatures plus a byte-for-byte confirmation; the exact order
   is **no verificado**). Kongtrol does the same, narrowed further: same base name, then same size,
   then SHA-256 of that file only. A SHA-256 match of the whole file is equality for any practical
   purpose; the stored copy is still built from the dropped bytes, so a collision could only pick
   the folder, and that folder's files then pass the ADR-0105 checks.
3. **Where to look.** OpenVPN GUI's per-user folder `%USERPROFILE%\OpenVPN\config` (the owner's
   case) and `%ProgramFiles%\OpenVPN\config`; Downloads, Desktop, Documents from the Known Folder
   API (it follows a folder moved into OneDrive); on Linux the XDG user dirs, `~/.config/openvpn`,
   `/etc/openvpn/client`, `/etc/openvpn` (only when readable); on macOS Downloads, Desktop,
   Documents, Tunnelblick's private `Configurations` and Viscosity's `OpenVPN` folders (paths from
   knowledge, **no verificado**).
4. **Bounds.** A depth of 3 covers OpenVPN GUI's one-folder-per-profile layout and Tunnelblick's
   `X.tblk/Contents/Resources/`. 20 000 entries and 1 s keep a huge Downloads folder from stalling
   the import; 32 hashes at most. Links and junctions are never followed (Go 1.23+ marks a junction
   `ModeIrregular`); OneDrive cloud files are reparse points too (`IO_REPARSE_TAG_CLOUD*`): their tag
   is read with `FILE_FLAG_OPEN_REPARSE_POINT` and `FileAttributeTagInfo` without opening the data.
   The value `0x9000001A` and the "bits 12–15 vary" mask come from `winnt.h` from memory:
   **no verificado** (if wrong, OneDrive files are skipped, never followed). Network drives
   (`DRIVE_REMOTE`; Linux statfs NFS/SMB/CIFS/AFS/Ceph/9P/FUSE magic numbers; macOS `smbfs`, `nfs`,
   `afpfs`, `webdav`) are never searched.
5. **Absolute references.** OpenVPN GUI reads an absolute `ca C:\\…\\ca.crt` with the user's
   rights; the daemon and the CLI run as the same user, so reading it crosses no privilege
   boundary. What a hostile `.ovpn` could try is make Kongtrol copy an unrelated user file into
   the stored profile, or leak something to the VPN server. From `ovpnconf.go`: every file
   directive's content is inlined into its own block (`<ca>`, `<cert>`, `<key>`, `<tls-auth>`,
   `<tls-crypt>`, `<tls-crypt-v2>`, `<secret>`, `<dh>`, `<extra-certs>`, `<crl-verify>`, `<pkcs12>`
   base64). In TLS only the client certificate chain (`cert`, `extra-certs`, or the certificates in
   `pkcs12`) is sent, and OpenSSL must parse it as X.509 first; `key` is used to sign, `tls-auth` /
   `tls-crypt` / `secret` are keys for HMAC or encryption, `ca` / `crl-verify` / `dh` stay local.
   So the leak path is "a certificate the user already has", which is public by design. Remaining
   risks and their answer: (a) a UNC path (`\\attacker\s\ca.crt`) would make Windows send NTLM
   credentials (T1187): never opened (`CleanLocalPath` refuses `\\`, `\??\`, device paths; mapped
   network drives refused); (b) `/proc`, `/dev`, FIFOs: refused (path and file type checks); (c)
   copying unrelated secrets (an OpenSSH key, a token file) into `~/.kongtrol/configs` (0600, same
   user, still worth avoiding): outside the folder only PEM blocks an OpenVPN directive takes are
   accepted (`LooksLikeKeyFile`), so an OpenSSH key, a PGP key or a text token is refused.
6. **`../` references.** OpenVPN resolves them like any relative path after `--cd`; refusing them
   gave nothing that the absolute case does not already allow, so they get the same outside checks.
7. **Folder dialog.** zenity's `Directory()` option switches the same native dialog to folder mode
   (pkg.go.dev, ADR-0105's source); nothing new to vet.

## Manual test (owner)

1. Drop `C:\Users\<you>\OpenVPN\config\rluna.conf` on the panel's "Importar un archivo de VPN":
   it must import and say "Encontré los certificados junto a rluna.conf en C:\…\OpenVPN\config".
2. Copy only the `.conf` to a temporary folder outside the searched ones and drop it: the banner
   "Faltan 4 archivos de este .ovpn" lists them as chips, with "Elegir la carpeta donde están"
   under the text; pick `OpenVPN\config`: it imports.
3. With a Desktop or Documents in OneDrive (Files On-Demand), drop a copy of a file kept there:
   whether it is found is **no verificado**.
