# Startup cancellation and native menus

Accessed 2026-11-25.

- [Go: Concurrency Patterns: Context](https://go.dev/blog/context): cancellation
  propagates to derived contexts; work should stop when cancellation is signaled.
- [Apple: Menus](https://developer.apple.com/design/human-interface-guidelines/menus.md):
  menu labels should describe their action, and an ellipsis signals a further
  choice. A native menu is not an editable search interface.
- `fyne.io/systray` exposes menu item changes and clicks but no portable
  menu-open notification in the API used by this project. Reassigning slots
  during a poll could therefore change what a click will act on.

The startup queue and its ordering, cancellation, and client snapshot are
covered by automated tests; native menu behavior on each desktop is not
verified by those unit tests.
