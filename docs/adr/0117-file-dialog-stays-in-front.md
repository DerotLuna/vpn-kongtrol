<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0117: The native file dialog stays in front while open

Version 1.0.0 · 2026-11-25

Status: Accepted · Date: 2026-11-25 · Decision-makers: Ruben Luna (owner, real case), implemented by agent

Refines [ADR-0105](0105-import-ovpn-with-referenced-files.md) and [ADR-0112](0112-find-a-dropped-ovpns-files-automatically.md)
(folder mode). Research: none new.

## Context and Problem Statement

"Elegir archivo" opens the OS's own dialog from the daemon (`internal/filepicker`, zenity). The
daemon is a hidden background process: Windows refuses it the foreground while the user works in the
browser (`SetForegroundWindow` rules), so ADR-0105 moved the dialog to the top of the z-order once
(topmost and straight back). In the owner's test the dialog appeared and fell behind the browser at
once; it stayed open, the one-dialog lock stayed held, and the next click answered
`api.err.picker_busy` with no dialog in sight.

## Decision Drivers

- The dialog the user asked for must be visible until they pick or close it.
- Never steal focus from other apps beyond the dialog the user just requested.
- One dialog at a time stays (ADR-0105).
- Closing the panel's sheet must close the dialog too.

## Considered Options

1. Keep the single raise (ADR-0105 as is).
2. **Chosen:** keep the dialog topmost for its whole life (re-asserted every 250 ms without
   activating it; the foreground is requested once when it appears), and a second request while it is
   open brings it to the front before answering 409.
3. Synthetic input or `AttachThreadInput` to force the foreground: works around Windows' focus rules
   in ways Microsoft documents as unsupported and that also affect other apps.

## Decision Outcome

Chosen: option 2.

- `raise_windows.go`: only a window of this process with the dialog's exact title is touched;
  `SWP_NOACTIVATE` re-assertion until the dialog closes or the request ends. `bringToFront` restores a
  minimized dialog, sets it topmost and requests the foreground.
- `filepicker.go`: the open dialog's title is kept while it is shown; a busy `Pick`/`PickFolder`
  brings that dialog forward and still returns `ErrBusy`. `api.err.picker_busy` now says it was
  brought to the front (taskbar hint kept: Windows may still refuse the foreground).
- The panel aborts the pick request when the Add sheet closes; the handler's `r.Context()` ends and
  zenity closes the dialog (zenity v0.10.15 `file_windows.go` / `util_windows.go`, read in source).
- Other OSes: no change (`bringToFront` is a no-op).

### Consequences

- Good: the dialog cannot hide behind the browser; a second click finds it.
- Neutral: while open, the dialog stays above other windows (it is the task the user started).

### Confirmation

Go: `internal/filepicker/filepicker_test.go` `TestPick_BusyBringsTheOpenDialogToFront`.

**No verificado:** behaviour on a real Windows desktop and the fetch abort end to end (owner's manual test).
