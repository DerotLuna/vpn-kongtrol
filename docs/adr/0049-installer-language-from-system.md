<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0049: The Windows installer takes the system language, asks only for other languages

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

The Inno Setup installer showed the language dialog on every run, including updates, where
ADR-0046 already skips every other question. Kongtrol itself picks the language from the system
(AGENTS.md, i18n), so the question was unnecessary for Spanish and English systems.

## Considered Options

1. Keep the dialog (default `ShowLanguageDialog=yes`).
2. **`ShowLanguageDialog=auto`.**
3. `ShowLanguageDialog=no` (always the first language, never ask).

## Decision Outcome

Option 2. Inno Setup docs: "When set to auto, the dialog will only be displayed if Setup does not
find a language identifier match"
(https://jrsoftware.org/ishelp/topic_setup_showlanguagedialog.htm, accessed 2026-11-19). Spanish
and English systems get no question, on first install and on updates; other system languages
still choose. `UsePreviousLanguage` (default yes) keeps the previous install's language as the
default (https://jrsoftware.org/ishelp/topic_setup_usepreviouslanguage.htm, accessed
2026-11-19); whether it overrides the system match when the dialog is skipped is not stated
there: **no verificado**, check on the next release candidate.

### Consequences

- Good: an update asks nothing it already knows; matches how the app picks its language.
- Bad: someone on a Spanish system who wants an English installer cannot pick it (the app
  language can still be changed with `kongtrol config lang`).
