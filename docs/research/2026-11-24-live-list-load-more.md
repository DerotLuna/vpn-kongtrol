<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Live lists: load more, filters and held updates instead of pages

Version 1.0.0 · 2026-11-24

Backs [ADR-0106](../adr/0106-recent-list-load-more-and-held-updates.md). Sources accessed 2026-11-24.

## Question

Tráfico › Recientes shows up to 500 destinations that refresh every 15 s, newest first. Rendering
them all is a very long list; pages would shift under the reader on every refresh (an entry seen
again moves to the top). What do established guidelines say?

## Findings

- NN/g, Tim Neusesser, "Infinite Scrolling: When to Use It, When to Avoid It" (2022-09-04),
  https://www.nngroup.com/articles/infinite-scrolling-tips/: "Load more" is presented as a middle
  ground between pagination and infinite scroll: it stops the endless flow, keeps the footer
  reachable, at the cost of one click; with infinite scroll "it is hard to remember the location
  of any specific item and return to it". No option is best for every case. The article does not
  address lists that change while being read.
- MDN, `overflow-anchor`, https://developer.mozilla.org/en-US/docs/Web/CSS/overflow-anchor:
  scroll anchoring "adjusts scroll position to minimize content shifts"; `none`: "The element
  won't be selected as a potential anchor." Used on the "Ver más" row so appending rows above it
  never scrolls the page.
- MDN, ARIA `status` role,
  https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/status_role:
  "Elements with the role status have an implicit aria-live value of polite and an implicit
  aria-atomic value of true." The "N nuevas · Mostrar" notice lives in one. That the region must
  already be in the DOM before its text changes is common practice, not stated on that page
  (**no verificado** there); the panel keeps it mounted.
- The "N new posts · Show" pattern (new items held at the top of a timeline until the user asks)
  is widely used by social feeds and GitHub's activity views; **no verificado** against official
  documentation of those products.

## Conclusion

First 10 rows, then "Ver 20 más (quedan N)"; client-side search and "Salió por" filters over the
whole list (500 entries at most, so no API change); the rows on screen are a snapshot whose data
updates in place, and new entries wait behind a polite "N nuevas · Mostrar".
