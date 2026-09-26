# 7. Farm/Cycle/Listing CRUD: ownership chain, deletion invariants, cycle state machine

## Status

Accepted

## Context

Phase 2 of the roadmap calls for complete CRUD on `farm`, `cycle` and
`listing` with validation rules (dates, quantities), plus two
invariants stated explicitly: a cycle without a listing breaks nothing,
and a listing without a cycle is impossible. None of these three
entities carry an owner-facing concept of their own beyond `Farm.OwnerID`
— a `Cycle` only knows its `FarmID`, and a `Listing` only knows its
`CycleID` — so authorizing a write to a cycle or a listing means
resolving ownership through a chain (`listing → cycle → farm → owner`),
not a direct field comparison.

## Decision

**Ownership resolution.** `application/usecase/cycle` and
`application/usecase/listing`'s write usecases look up the owning farm
through the reference chain and compare `Farm.OwnerID` against the
caller's token `UserID` — `domain.ErrForbidden` on a mismatch. This is
duplicated logic (a helper, `ownerOf`, factors out the listing → cycle →
farm chain within package `listing`) rather than a shared abstraction
across packages, since the chain differs by one hop between the two
modules and a shared "resolve owner" interface would need to know about
both aggregates anyway.

**Deletion invariants, enforced going both directions.** The DoD states
one direction explicitly (a cycle with no listing is fine); the reverse
is just as necessary for data integrity and wasn't left as an implicit
assumption:

- `DeleteFarm` refuses (`domain.ErrConflict`) if the farm still has any
  cycles — deleting it would orphan their `FarmID` reference.
- `DeleteCycle` refuses (`domain.ErrConflict`) if the cycle still has a
  listing (checked via a new `ListingStore.FindByCycleID`) — deleting it
  would orphan the listing's `CycleID` reference, and any reservation
  that references that listing.
- Deleting a listing is never a hard delete — see the "no listing
  delete" bullet below.

Both checks live in the usecase, not the repository: a repository
should only ever touch its own collection (`FarmRepository` never
queries `cycles`), so the invariant is the usecase's job to enforce by
calling both ports it depends on.

**Cycle status is a state machine, not a free-form field.**
`UpdateCycleStatus` only allows `planned → in_progress`, `planned →
cancelled`, `in_progress → done`, and `in_progress → cancelled` — never
skipping `in_progress`, and never leaving a terminal state (`done`,
`cancelled`). This is the "règles de validation" the DoD asks for
beyond dates and quantities: a cycle jumping straight from `planned` to
`done` would be a bug the type system can't catch on its own, since
`CycleStatus` is just a string.

**`AvailableQuantity` starts equal to `Capacity`, set once, at
creation.** Nothing after `CreateCycle` ever recomputes it except
`application/usecase/reservation`'s atomic decrement (ADR-0001) — this
usecase writes it exactly once and never touches it again, keeping the
"who's allowed to change this field" boundary unambiguous.

**No hard delete for a listing.** `UnpublishListing`, not
`DeleteListing`, sets `Status` to `domain.ListingUnpublished`. A
reservation stores its `listingId` directly (no embedding, per ADR-0001)
— hard-deleting a listing after a reservation exists against it would
break that reference. This mirrors the reasoning `infrastructure/mongodb`
already applies elsewhere (nothing in this codebase hard-deletes a
document another collection references by ID).

**Two ports per shared aggregate, not one widened interface.**
`ports.CycleRepository` (reservation's one-method slice) and
`ports.CycleStore` (the fuller CRUD port `application/usecase/cycle`
depends on) are separate interfaces — same for `ListingRepository` /
`ListingStore` — both pairs implemented by the same
`infrastructure/mongodb` concrete type. Widening the original interface
instead would have forced `reservation`'s existing one-method fake to
grow methods it never calls, the exact premature coupling ADR-0002
already argues against.

**Role restriction only at creation, not at every write.**
`POST /farms`, `POST /cycles` and `POST /listings` require
`domain.RoleFarmer` (`interfaces/httpserver.RequireRole`); updates,
deletes and status transitions rely on the ownership check alone. A
buyer can never own a farm to begin with (self-registration as a
farmer is the only way one gets created), so a second role gate on
every mutating endpoint would be redundant — the ownership chain
above already rejects a buyer's token on any of those routes.

## Consequences

**Easier:** the two-directions-of-deletion-safety invariant and the
cycle state machine are both real, testable rules with cheap unit tests
(`application/usecase/farm`, `.../cycle`) — not just documentation.

**Harder:** three extra round trips (farm existence, cycle existence,
ownership) before most cycle/listing writes succeed — acceptable at
this scale, would be worth reconsidering (e.g. denormalizing an
`ownerId` onto `Cycle`/`Listing`) only if profiling ever showed it
mattered, which nothing here suggests yet.

**Explicitly not doing:** a generic "resolve owner of any aggregate"
abstraction, or moving these two ports into one — see the "two ports"
decision above; and no soft-delete/archive state for `Farm` or `Cycle`
(only `Listing` needs it, since only listings are referenced by
reservations).
