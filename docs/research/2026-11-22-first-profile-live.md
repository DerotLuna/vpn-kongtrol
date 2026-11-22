# Research: First profile and tray menu transition

Sources checked 2026-11-22.

- [Fyne systray 1.12.2 source](https://github.com/fyne-io/systray/blob/v1.12.2/systray.go)
  shows that ResetMenu removes items and Remove closes the item's ClickedCh.
  Kongtrol's temporary exit item read once from that channel before quitting.
- [Go's channel receive specification](https://go.dev/ref/spec#Receive_operator)
  explains why a single receive wakes when the channel closes. Ranging over
  actual click events ends the handler on close.
- [CSS Grid track sizing](https://www.w3.org/TR/css-grid-2/#algo-flex-tracks)
  describes how equal flexible rows size to their largest content contribution.
  Detection tiles stretch inside each row and can still wrap on small screens.

Kongtrol already creates its adapter registry and runtime services in setup
mode. The special case skipping live sync was the reason the first save
requested a daemon restart and invalidated the browser session.
