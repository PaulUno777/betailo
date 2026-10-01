# Architecture Decision Records

An ADR captures one significant technical decision, the context that
forced it, and what it costs — written once, close to the moment the
decision was made, and never rewritten afterward. When a later decision
overturns an earlier one, a new ADR supersedes it (link both ways); the
old one stays as a record of what used to be true and why it changed.

This is deliberately separate from the two living reference documents:

- **"Plateforme Éleveurs — Proposition d'application Go (MVP Portfolio)"**
  is the product reference: scope, features, roadmap. It answers *what*
  and *why build it at all*.
- **"Betailo — Concepts & Architecture"** is the technical reference: data
  model, sequence diagrams, deployment topology. It answers *how it
  works*, as a snapshot of the current design.
- **An ADR** answers *why this specific technical choice, at this
  specific point, over the alternatives on the table* — the reasoning
  the other two documents assume but don't argue for. Read an ADR when
  you're about to revisit a decision and need to know whether the
  reasons that made it still hold.

## Format

Each ADR is `NNNN-short-title.md`, numbered sequentially, in this repo's
root-level `docs/adr/`. Status is one of `Proposed`, `Accepted`,
`Superseded by ADR-NNNN`, or `Deprecated`. Sections: **Context**, **Decision**,
**Consequences** (what gets easier, what gets harder, what we're
explicitly not doing).

## Index

| ADR | Title | Status |
| --- | --- | --- |
| [0001](0001-mongodb-over-postgresql.md) | MongoDB over PostgreSQL for persistence | Accepted |
| [0002](0002-layered-clean-architecture.md) | Layered Clean Architecture (domain / application / infrastructure / interfaces) | Accepted |
| [0003](0003-seaweedfs-for-local-object-storage.md) | SeaweedFS, not MinIO, for local S3-compatible storage | Accepted |
| [0004](0004-mongo-single-node-replica-set-for-local-dev.md) | Single-node MongoDB replica set for local dev transactions | Accepted |
| [0005](0005-english-only-in-code-french-in-product-docs.md) | English exclusively in code; French stays the product-documentation language | Accepted |
| [0006](0006-stateless-jwt-auth.md) | Stateless JWT auth: bcrypt + HMAC, no session store, no refresh tokens | Accepted |
| [0007](0007-farm-cycle-listing-crud-invariants.md) | Farm/Cycle/Listing CRUD: ownership chain, deletion invariants, cycle state machine | Accepted |
| [0008](0008-refresh-token-rotation-httponly-cookie.md) | Refresh-token rotation in an httpOnly cookie | Accepted |
