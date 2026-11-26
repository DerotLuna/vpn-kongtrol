# Rule sheet and suggestion navigation

Accessed 2026-11-26.

- [Chrome natural interactions](https://developer.chrome.com/docs/modern-web-guidance/guides/user-experience/natural-interactions)
  recommends native horizontal scrolling and focusable controls for chip rows.
- [Chrome scrollability affordance hints](https://developer.chrome.com/docs/modern-web-guidance/guides/ui-atoms/scrollability-affordance-hints)
  identifies a visible scrollbar as the primary cue, with fades optional.
- [W3C carousel pattern](https://www.w3.org/WAI/ARIA/apg/patterns/carousel/)
  describes accessible controls for slide carousels; this list is instead a
  native scroller of individual action buttons, with no automatic rotation.

The suggestions are actions, not selected filters: pressing one appends
destinations without removing prior input. Keeping the list in the rule editor
avoids presenting the same action before its destination field exists.
