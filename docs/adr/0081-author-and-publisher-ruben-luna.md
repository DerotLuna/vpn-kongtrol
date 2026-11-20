<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0081: The author and publisher is Rubén Luna, everywhere

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Windows "Installed apps" listed the publisher as "Kongtrol", as if it were a company; it is a
free, open-source project by one person. Package metadata was inconsistent ("Kongtrol" in the
installer, `.deb`/`.rpm` and winget; "DerotLuna" in the AUR) and no file named the copyright holder,
which the Apache License 2.0 appendix recommends (https://www.apache.org/licenses/LICENSE-2.0#apply,
accessed 2026-11-20). Single-author open-source programs commonly list the author as publisher
(PuTTY "Simon Tatham", 7-Zip "Igor Pavlov"; no verificado for every distribution channel).

## Decision Outcome

The owner chose "Rubén Luna" (2026-11-20), in every place an author or publisher appears:

- Windows installer: `AppPublisher=Rubén Luna`, `AppCopyright=Copyright 2026 Rubén Luna`
  (`packaging/windows/kongtrol.iss`, UTF-8 without BOM — supported since Inno Setup 6.3,
  https://jrsoftware.org/files/is6.3-whatsnew.htm; the release uses 6.7.4).
- `.goreleaser.yaml`: nfpm `vendor` and `maintainer`, winget `publisher`, AUR `maintainers`
  (`Rubén Luna (DerotLuna)`); contact is the GitHub profile, never an email.
- npm packages `author`, macOS `NSHumanReadableCopyright` (`packaging/macos/Info.plist`).
- `NOTICE` at the repository root ("Copyright 2026 Rubén Luna"), shipped in the archives and in
  `/usr/share/doc/kongtrol/` next to `LICENSE`.
- Landing footer: "Hecho con ♥ por Rubén Luna" (EN "Made with ♥ by Rubén Luna"), linking to the
  GitHub profile; the heart is a gold mark with an accessible name.

The name in these texts is display only: the publisher Windows verifies (SmartScreen, UAC) comes
from a code-signing certificate (issue #30), not from these fields.

## More Information

- Docs: `README.md`, `DESIGN.md` (Footer).
