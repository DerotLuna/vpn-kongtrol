# ADR-0123: Suggested sites in profile and rule flows

Status: Superseded by ADR-0124 · Date: 2026-11-26 · Decision-maker: owner

Research: [site suggestions and access restrictions](../research/2026-11-26-venezuela-site-suggestions.md).
Extends ADR-0122.

## Decision

After saving either kind of profile, a suggested site starts an editable,
unsaved rule for that profile. Rule creation and editing both provide the same
ordered suggestions: Claude, ChatGPT/OpenAI, X, Reddit, Telegram Web, Canva and
Punto de Corte. Each action appends destinations without replacing existing
ones or duplicating recognized domains; multiple actions can build one rule.
Existing rules change only when saved; new rules remain paused until activated.

The examples reflect documented restrictions, not a claim that a service is
currently blocked or that its entire traffic is routed. The interface warns
about incomplete coverage and provider terms. Nothing checks availability or
contacts external services automatically.
