# ADR-0126: Oracle and Java rule suggestion

Status: Accepted · Date: 2026-11-26 · Decision-maker: owner

Supersedes ADR-0124. Research: [Oracle and Java domains](../research/2026-11-26-oracle-java-suggestions.md)
and [sheet navigation](../research/2026-11-26-rule-sheet-navigation.md).

## Decision

Use "Agregar regla de salida" consistently for the action and form. After
either profile creation path, that action opens the rule form for the saved
profile. Back returns to the previous result, preserving the unsaved draft.
The rule form is wider on desktop but remains constrained by the viewport.

Rule creation and editing show one horizontally scrollable row of suggestion
buttons immediately before the destination field: Claude, ChatGPT/OpenAI,
and Oracle/Java. Remove X, Reddit, Telegram Web, Canva and Punto de Corte as
requested by the owner; old reports do not establish current restrictions.
Each action appends editable destinations without duplication, saving or
activating anything. A visible scrollbar and keyboard-focusable buttons
provide access without requiring a swipe or search.

Restrictions vary, and suggestions do not guarantee service coverage or
access. Oracle and Java access restrictions in Venezuela are not verified;
users must respect provider terms. The app does not check availability or
contact external services automatically.
