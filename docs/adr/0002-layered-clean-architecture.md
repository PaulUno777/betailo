# 2. Layered Clean Architecture (domain / application / infrastructure / interfaces)

## Status

Accepted

## Context

The backend's first cut of the MongoDB migration organized code by
business module (`internal/user`, `internal/farm`, `internal/reservation`,
...), each holding a domain struct with `bson`/`json` tags directly on
it, plus a thin repository wrapping a `*mongo.Collection`. That's a
common, defensible Go layout, but it has two properties this project
explicitly wanted to avoid once `reservation` became the first module
with real business logic (a MongoDB transaction preventing overselling):

- The domain struct's tags made it depend on the persistence framework
  by construction — a `domain.Reservation` was also, unavoidably, a
  MongoDB document shape.
- The transaction logic lived directly on a type that also held a
  `*mongo.Client`, so the business rule ("check-and-decrement and
  insert happen atomically") could only be tested against a real
  MongoDB instance.

## Decision

Restructure into a layout with one-way dependencies — `interfaces` →
`application` → `domain`, and `infrastructure` implements
`application/ports` without `domain`/`application` ever depending back
on it:

- **`domain/`** — plain Go structs, no `bson`/`json` tags, no import of
  `mongo-driver` or any other framework. `bson.ObjectID` is the one
  exception, kept because the ID *type* the "Concepts & Architecture"
  document mandates (an `ObjectId` primary key per collection) is a
  value type, not a driver/connection type — the same category of
  compromise as using `uuid.UUID` in a domain layer that otherwise
  avoids third-party types.
- **`application/ports/`** — interfaces declared where they're
  consumed, each reduced to the methods actually called (Interface
  Segregation): `ListingRepository.CycleAndPrice`,
  `CycleRepository.DecrementAvailability`, `ReservationRepository.Insert`,
  `TxManager.WithinTx`, `ObjectStorage.PresignUpload/PresignDownload`.
- **`application/usecase/<name>/`** — one usecase per real business
  rule, as a struct holding its ports plus an `Execute(ctx, Input) (*Output, error)`
  method. `ReserveCycle` is the only one that exists today; the pattern
  is deliberately not applied to modules with no usecase yet (see
  Consequences).
- **`infrastructure/mongodb/`** — the only package that imports
  `mongo-driver`. Each repository implements one `application/ports`
  interface and converts between the tag-free `domain` entity and a
  private, `bson`-tagged document struct (e.g. `reservationDoc`) —
  mapping lives here, never on the domain type.
- **`infrastructure/objectstorage/`** — implements `ObjectStorage`
  against any S3-compatible endpoint.
- **`internal/app/`** — the composition root: the one place that
  constructs the concrete infrastructure and wires it into usecases
  (`Initialize`), separately from running the schema bootstrap
  (`Bootstrap`) and serving (`Run`). `cmd/api/main.go` only calls those
  three methods.

This mirrors the layering already used in this account's other Go
services (`jubely-saas/backend`), which additionally validated the
convention: exported struct fields for infrastructure dependencies
(`TxManager{Client: ...}`), not constructor functions; one method per
port file; usecases named as verb-phrase structs.

## Consequences

**Easier:**
- `ReserveCycle.Execute` is unit-tested with four in-memory fakes and
  zero MongoDB dependency (`reserve_cycle_test.go`) — this is the
  concrete payoff the restructure was for.
- Swapping MongoDB for something else (or R2 for a different object
  store) touches `infrastructure/`, never `application/` or `domain/`.
- The dependency direction is enforced by package boundaries: a
  `domain` import cycle back to `infrastructure` is a compile error, not
  a code-review rule someone has to remember.

**Harder:**
- More files and more indirection for what is still, today, one real
  usecase — `ReserveCycle` alone touches nine files across four
  directories where the module-per-package version needed two.
- The mapping between `domain.Reservation` and `reservationDoc` (and
  the enum-value contract with `schemas/*.json`, see ADR-0001) is a
  second place a field rename has to happen.

**Explicitly not doing:** applying ports + usecase structure to modules
with no real business logic yet (`user`, `farm`, `review`, `forum`,
`report`/trust) — they stay as bare `domain` structs. Adding a
`Repository` interface with methods nobody calls, or a usecase that
just forwards to a repository, would be abstraction with no consumer to
justify it. Each of those modules gets its own `application/ports` +
`infrastructure/mongodb` entries the same way `reservation` did, when a
real rule (not just CRUD) shows up for it — see the roadmap in "Betailo
— Plan de développement".
