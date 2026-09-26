# 4. Single-node MongoDB replica set for local dev transactions

## Status

Accepted

## Context

`application/usecase/reservation.ReserveCycle` runs its capacity check
and reservation insert inside a MongoDB session/transaction
(`application/ports.TxManager`, ADR-0001) — that's the mechanism that
makes overselling impossible under concurrent reservations. MongoDB
multi-document transactions only work on a replica set (or a sharded
cluster); a standalone `mongod` refuses them outright.

`docker-compose.yml`'s original `mongo` service ran the stock `mongo:7`
image with no replica-set configuration — a standalone server. This
went unnoticed through the initial migration because nothing had
exercised a real transaction yet: `go build`/`go vet`/`go test -race`
all pass against fakes (see ADR-0002), and `/health` doesn't touch
Mongo transactionally. It surfaced only when `ReserveCycle` was run
against the live container for the first verification pass:

```
MongoServerError: Transaction numbers are only allowed on a replica set
member or mongos
```

Confirmed at the `mongosh` level first (a transaction against an
unconfigured `mongo:7` fails with exactly that error; the identical
transaction against a `--replSet`-initiated instance succeeds), then
end-to-end through the real Go code path.

## Decision

Run `mongo` with `--replSet rs0`, and have its healthcheck both wait for
`mongod` and initiate the replica set on first success:

```yaml
healthcheck:
  test: >
    mongosh --quiet --eval "try { rs.status().ok } catch (e) {
    rs.initiate({_id:'rs0',members:[{_id:0,host:'mongo:27017'}]}).ok }"
```

This needs no separate init container: the healthcheck is idempotent
(`rs.status().ok` short-circuits once initiated) and `backend` already
depends on `mongo` being `service_healthy`, so it never starts before
the replica set exists. `MONGODB_URI` gains `?replicaSet=rs0`.

This mirrors Atlas's own free tier, which is *always* a replica set —
so this isn't local dev diverging from production, it's local dev
matching a constraint production already has (see ADR-0001).

One consequence needed its own fix: the replica set advertises its
member as `mongo:27017`, a name that only resolves inside the
docker-compose network. Any client connecting from the host — MongoDB
Compass, `mongosh` run outside Docker, or `go run ./cmd/api` outside the
container — reaches `mongo:27017` for the initial handshake (the
port is published to the host) but then fails trying to resolve
`mongo` for topology-aware operations. The fix is `directConnection=true`
in the connection string, which skips replica-set discovery and talks
to that one node directly (correct here since it's always the primary
on a single-node set). Documented in `backend/.env.example` and the
README; not needed for `backend` itself, which resolves `mongo` fine
from inside the compose network.

## Consequences

**Easier:** the local stack now genuinely exercises the same transaction
guarantee production has, instead of silently allowing something Atlas
would reject.

**Harder:** `docker-compose down -v` followed by `up` re-initiates the
replica set from scratch every time (harmless, just a few extra seconds
on the healthcheck's first retries). Anyone connecting a GUI tool or a
host-side script to `mongo` for the first time hits the
`directConnection=true` requirement without warning unless they've read
this ADR or the README.

**Explicitly not doing:** a multi-node replica set locally — one node
is enough to unlock transactions and matches what a single Atlas free-tier
cluster looks like from the driver's perspective; more nodes would only
be useful for testing failover, which isn't a concern this project has
yet.
