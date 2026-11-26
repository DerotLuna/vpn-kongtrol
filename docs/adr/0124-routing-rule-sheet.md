# ADR-0124: Routing rule sheet and suggestions

Status: Superseded by ADR-0126 · Date: 2026-11-26 · Decision-maker: owner

Supersedes ADR-0123. Research: [site restrictions](../research/2026-11-26-venezuela-site-suggestions.md)
and [sheet navigation](../research/2026-11-26-rule-sheet-navigation.md).

## Decision

Use "Agregar regla de salida" consistently for the action and form. After
either profile creation path, that action opens the rule form for the saved
profile. Back returns to the previous result, preserving the unsaved draft.
The rule form is wider on desktop but remains constrained by the viewport.

Rule creation and editing show one horizontally scrollable row of suggestion
buttons immediately before the destination field. Claude comes first, then
ChatGPT/OpenAI, then X, Reddit, Telegram Web, Canva and Punto de Corte.
Each action appends editable destinations without duplication, saving or
activating anything. A visible scrollbar and keyboard-focusable buttons
provide access without requiring a swipe or search.

Restrictions vary, and suggestions do not guarantee service coverage. The app
does not check availability or contact external services automatically.
