# 1. MongoDB over PostgreSQL for persistence

## Status

Accepted

## Context

The backend originally shipped with a PostgreSQL schema (`backend/migrations/`,
`database/sql` + `pgx`). The design changed to MongoDB Atlas in production,
with an S3-compatible object store for listing photos (Cloudflare R2 in
production, a local S3-compatible gateway in dev), before any business
logic beyond scaffolding existed — see "Betailo — Concepts & Architecture"
and "Plateforme Éleveurs — Proposition d'application Go" for the full
technical and product rationale this ADR summarizes rather than repeats.

The two facts driving the choice:

- `Reservation.quantity` against `Cycle.availableQuantity` is the one
  place this domain needs a hard, low-level correctness guarantee
  (never oversell a cycle's capacity under concurrent reservations).
  MongoDB's atomic `findOneAndUpdate` with a `$gte` guard on the field
  being decremented gives that guarantee at the single-document level,
  with no explicit lock.
- Three relationships in the data model are naturally embedded and
  always read/written with their parent, never independently:
  `Cycle.healthLog`, `Reservation.payment`, `ForumPost.replies`. A
  document store expresses "always read/written together, bounded
  size" directly; a relational schema would model the same thing as a
  child table joined on every read for no benefit.

## Decision

Use MongoDB (Atlas in production, a local `mongo` container in dev) via
the official `mongo-driver/v2`, no ODM, explicit queries in the
infrastructure layer. Multi-document transactions (reservation's
capacity check + insert) run inside a session/transaction — supported
on Atlas's free tier because it is always a replica set (see ADR-0004
for what that requires locally). Schema constraints move from SQL
migrations to one `$jsonSchema` validator per collection
(`backend/schemas/`), applied idempotently at startup instead of via a
migration tool.

Object storage for listing photos is a separate concern from the
database: `application/ports.ObjectStorage`, implemented via
`aws-sdk-go-v2` against any S3-compatible endpoint (Cloudflare R2 in
production; see ADR-0003 for the local dev choice).

## Consequences

**Easier:**
- The oversell guard is one atomic operation instead of an explicit
  `SELECT ... FOR UPDATE` + application-level retry loop.
- Three relationships that were previously separate tables with a
  foreign key are now embedded arrays, with no join needed to read them.
- Schema changes are a JSON file edit + restart, not a migration file
  plus an `up`/`down` pair.

**Harder:**
- No foreign keys: `Report.targetId`/`targetType`'s referential
  integrity, and `Listing.cycleId`'s optionality, are guaranteed by the
  usecase layer, not the database. A bug there fails silently instead
  of at the database layer.
- Multi-document transactions require a replica set even for a single
  node — see ADR-0004 for the concrete local-dev consequence this had.
- `$jsonSchema` validators and the Go domain types that must match them
  (`domain.CycleStatus`, `domain.Role`, etc.) are two independent
  sources of truth kept in sync by convention and a doc comment on each
  Go type, not by a generator. A future contributor changing one
  without the other reintroduces exactly the kind of drift this ADR's
  own history (see the commit fixing English-only enum values) already
  had to catch once.

**Explicitly not doing:** an ODM (`mgm` or similar) or a query builder —
explicit `bson.M`/`bson.D` queries in `infrastructure/mongodb`, mirroring
the "no ORM" decision this project already made for the SQL era.
