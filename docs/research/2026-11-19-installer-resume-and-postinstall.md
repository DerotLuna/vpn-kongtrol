<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Installer: postinstall launch and reconnect after update

Version 1.0.0 · 2026-11-19 · Backs [ADR-0050](../adr/0050-installer-resume-tunnels-and-open-request.md)

## Questions

1. Does the [Run] entry `kongtrol-tray.exe … Flags: postinstall nowait skipifsilent` need more
   flags (`runasoriginaluser`, `WorkingDir`) to start the tray?
2. Can the installer run the NEW `kongtrol.exe` before the files are copied?
3. In what order do [Run] entries run?

## Findings (Inno Setup docs, accessed 2026-11-19)

Source: https://jrsoftware.org/ishelp/topic_runsection.htm

- "Programs are executed in the order they appear in the script. By default … Setup/Uninstall
  will wait until the program has terminated before proceeding to the next one, unless the
  nowait, shellexec, or waituntilidle flags are used." → `service install --if-outdated` then
  `resume` then the tray, in that order.
- `postinstall`: "Instructs Setup to create a checkbox on the Setup Completed wizard page."
- `runasoriginaluser`: "This is the default behavior when the postinstall flag is used." → not
  needed (and Kongtrol's installer is per-user, `PrivilegesRequired=lowest`, ADR-0025).
- `WorkingDir`: "If this parameter is not specified or is blank, it uses the directory from the
  Filename parameter" → `{app}`; the tray finds `kongtrol.exe` next to itself anyway.
- `skipifsilent`: skip the entry when Setup runs silently; entries without it (the `resume`
  entry) also run on silent installs.

Source: https://jrsoftware.org/ishelp/topic_isxfunc_extracttemporaryfile.htm

- "Extracts the specified file from the [Files] section to a temporary directory. To find the
  location of the temporary directory, use ExpandConstant('{tmp}'). The extracted files are
  automatically deleted when Setup exits." An exception is raised if the file is not found or not
  extracted → wrapped in `try … except` with a fallback to the installed copy.
- "When solid compression is enabled, be sure to list your temporary files at (or near) the top
  of the [Files] section." → `kongtrol.exe` is the first entry.

## Conclusion

The flags were already correct; the "nothing opens" bug was in the tray's start logic (one-time
marker + "no config" condition + single instance) and in choosing the checkbox by program files
instead of profiles. Reconnecting after an update needs the new binary before the copy
(`ExtractTemporaryFile`) and a [Run] entry after the helper update.

Reasoned, **no verificado** on a real install: the full Windows update flow (CI builds the
installer; not run end to end here).
