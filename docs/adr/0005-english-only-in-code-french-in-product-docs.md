# 5. English exclusively in code; French stays the product-documentation language

## Status

Accepted

## Context

The three reference documents in `docs/` (the proposal, "Concepts &
Architecture", and the development plan) are French — the product's
target market is French-speaking, and the documents describe *what* to
build and *why* in that language. Early Go source (comments, and in a
few cases enum string values) mirrored that: `domain.Role` used
`"eleveur"`/`"acheteur"`, `domain.CycleStatus` used
`"planifie"`/`"en_cours"`/... , matching the French vocabulary in those
documents' data model section.

This created a real bug, not just a style inconsistency: enum string
values are also MongoDB `$jsonSchema` validator contract (ADR-0001).
When some of those values were changed to English without the matching
`schemas/*.json` update happening at the same time, in the same pass,
every write of the untranslated collections would have failed
validation — caught before it shipped, but only because it was caught.
Partial translation of a two-sided contract (Go enum ↔ JSON schema enum)
is worse than either language, consistently applied, on both sides.

## Decision

Code is English, without exception: Go identifiers, comments, and
string literals (including enum values persisted to MongoDB). Product
documentation (`docs/*.md`, the three reference documents, and
`README.md`/`CONTRIBUTING.md`, which describe the product and how to
work on it) stays French, matching the existing convention and the
target market.

Concretely, for the domain vocabulary already in French in the
reference documents: `eleveur` → `farmer`, `acheteur` → `buyer`; status
enums translate to their plain English equivalent
(`planifie`/`en_cours`/`termine`/`annule` →
`planned`/`in_progress`/`done`/`cancelled`, and similarly for
`PaymentStatus`, `ReservationStatus`, `ReportStatus`). A comment on each
enum type in `domain/` names the `schemas/*.json` file its values must
stay in sync with, specifically to make the two-sided-contract risk
this ADR exists because of visible at the point someone is about to
change one side.

## Consequences

**Easier:** contributors who don't read French can work in `backend/`
without translating comments in their head first. Enum values read the
same whether you came from the Go code or the API response — no mental
mapping from a French wire value to an English concept.

**Harder:** the three French reference documents' data-model sections no
longer match the Go enum values verbatim (they still say `eleveur`,
`planifie`, etc.) — read status/role values here, in `domain/`, as the
current source of truth for the literal strings, and the French
documents for the concepts those strings name. Translating the domain
vocabulary loses a little of the original French-market framing in
code-level identifiers (`Farm`/`farmer` reads more generic than
`Farm`/`eleveur`, which was specific to the product's actual domain
term) — an acceptable trade for a codebase in one consistent language.

**Explicitly not doing:** translating the three reference documents or
the README/CONTRIBUTING into English — they document a French-market
product for a French-reading audience, and that's a separate decision
from what language the code itself is written in.
