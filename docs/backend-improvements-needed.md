# Backend improvements needed (found while building the frontend)

A living punch-list, not an ADR: gaps in the current API surface found while
wiring the Angular frontend against it. Nothing here has been acted on yet —
noted so they aren't lost, to fix once the backend work resumes.

## Review model is too thin for the design

`Review` is `{id, rating, comment}`. The design's `ReviewCard` calls for an
author name, verified-purchase reference (`RSV-xxxx`), photos, tags, and an
optional farmer reply — none of which the API returns today. `ReviewSummary`
also lacks the four per-criteria scores (santé, conformité, communication,
délais) the design's `RatingSummary` shows.

**Needed:** `authorId`/display name, the source `reservationId` (so the
frontend can render "Avis vérifié · RSV-xxxx" without inventing it),
optional photo URLs, optional tags, an optional farmer-reply field. Per-
criteria scores are a bigger addition (extra input fields, a new stored
shape) — lower priority than the rest.

## No public trust/verification signal outside the marketplace search join

`GET /listings` (search) returns `farmTrustScore` and `farmVerified` on each
result, computed server-side. `GET /farms/{id}` returns neither — so a
listing detail page reached directly (not from the search grid) can't show
the farmer's trust score or verified badge. The frontend currently omits
both there rather than fake them.

**Needed:** either add `trustScore`/`verified` to the plain `Farm` response,
or a small public `GET /farms/{id}/summary` that includes them alongside
review stats.

## `GET /reservations/mine` has no joined listing/farm summary

The response is bare `Reservation[]` (`listingId`, `buyerId`, amounts,
status — no listing title, photo, or farm name). Rendering a usable "my
reservations" list means either an N+1 fetch per row (one `GET
/listings/{id}` per reservation — what the frontend does today) or showing
raw IDs.

**Needed:** a joined summary similar to `MarketplaceListing`, or at minimum
`listingTitle`/`listingPhoto`/`farmName` denormalized onto the reservation
response.

## Admin surface needs backing data the current usecases don't compute

`Admin`'s system-health panel (API p95, worker goroutine count, outbox
queue depth) isn't backed by any endpoint today. (Moderation's risk score
and signal breakdown already are: `GET /admin/reports/{id}` returns
`riskScore` and `signals`.)
