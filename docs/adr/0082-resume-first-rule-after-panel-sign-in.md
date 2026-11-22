<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0082: Resume the first rule after signing in again

Version 1.0.0 · 2026-11-21

Status: Accepted · Date: 2026-11-21 · First-profile restart superseded by ADR-0092

## Context and Problem Statement

Saving the first profile in setup mode restarts Kongtrol. The new process rejects the previous
browser session because sessions live in memory. The Add sheet used that rejection as proof of
a successful restart and could send the user to Rules while the page was signed out.

## Decision Outcome

A restart that answers with an authorization refusal prompts the user to reopen the panel from
Kongtrol. The tab stops sending API requests and requesting live feed tickets after the refusal.
The recovery notice appears in the Add sheet when it is open and above the page otherwise.

The Add sheet preserves the saved profile name and the optional unsaved rule draft in the
browser's storage for the panel origin. A tab opened with a fresh local nonce can resume the
sheet for ten minutes. It removes the handoff when reading it. The sheet shows the destination
in the previous tab so it can be copied if browser storage is unavailable. Credentials and
connection files are excluded from the handoff. The replacement still requires a new session
using the existing nonce exchange from [ADR-0062](0062-origin-bound-sessions-token-rotation-listener-owner.md).

## Confirmation

The Add sheet test models a process that trusts a different page token after restart and verifies
that the user can save the recovered rule. Data layer tests check that a denied ticket stops
retries and a temporary outage continues to retry.

See [research on panel recovery](../research/2026-11-21-panel-session-recovery.md).
