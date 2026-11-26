# Profile lists and Windows picker cancellation

Accessed 2026-11-25.

- [Apple: Searching](https://developer.apple.com/design/human-interface-guidelines/searching.md)
  recommends a single global search and, where useful, a local search scoped to the current view.
- [Nielsen Norman Group: Filters and Sorting](https://www.nngroup.com/contents/self-paced-courses/filters-and-sorting-the-complete-design-guide/)
  recommends sorting defaults and labels that reflect how people browse.
- [Microsoft: WM_CLOSE](https://learn.microsoft.com/en-us/windows/win32/winmsg/wm-close)
  documents the close request sent to a window.
- `github.com/ncruces/zenity` v0.10.15 `file_windows.go` and `util_windows.go`:
  the library attempts cancellation through a COM call and a dialog hook.
  In a Windows desktop test, the dialog remained visible for two seconds but did
  not return within five seconds of context cancellation. Posting `WM_CLOSE` to
  the matching window of this process made the same test pass. This does not
  establish why the user's earlier dialog appeared to close.

The local history already stores `LastConnectedAt`; its age reflects the most
recent connection, including automatic reconnections, not a count of manual use.
Keep the list stable during interaction. A test file was selected in the native
Windows dialog; the full browser-to-daemon flow remains unverified.
