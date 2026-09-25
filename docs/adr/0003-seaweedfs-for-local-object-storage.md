# 3. SeaweedFS, not MinIO, for local S3-compatible storage

## Status

Accepted

## Context

The architecture document specifies MinIO as the local-dev counterpart
to Cloudflare R2 in production, both accessed through the same
S3-compatible `application/ports.ObjectStorage` interface. Standing up
`docker-compose.yml` with MinIO, `docker compose up --build` failed:

```
Image minio/minio:latest Pulling
Error pull access denied for minio/minio, repository does not exist or
may require 'docker login': denied: requested access to the resource is denied
```

The same failure reproduced for `minio/mc` (the CLI image originally
used to create the dev bucket), for an older pinned `RELEASE.*` tag on
Docker Hub, and for `quay.io/minio/minio` — ruling out a tag-specific or
registry-specific fluke. A generic Docker Hub pull (`alpine:latest`)
succeeded in the same environment in under a second, confirming the
restriction is specific to MinIO's own images, not a general registry
block.

Checking `minio/minio`'s own repository confirms why, precisely: it is
no longer maintained, and "the MinIO community edition is now
distributed as source code only. We will no longer provide
pre-compiled binary releases for the community version" — including
the Docker images. This isn't a login wall on an existing image; the
pre-built community images were discontinued outright, permanently,
in favor of MinIO's commercial AIStor product. The two paths their own
README gives for still running community MinIO are building the
binary from source (`go install github.com/minio/minio@latest`) or
building your own Docker image from their `Dockerfile` after that —
there is no pre-built image to pull, from any registry, going forward.

`chrislusf/seaweedfs:latest` pulled successfully in the same environment
and exposes an S3-compatible gateway (`weed server -s3`), making it a
drop-in replacement for the one thing local dev needs from this
service: an S3 API that `infrastructure/objectstorage.S3Provider`
(built against `aws-sdk-go-v2`, addressing everything through
`STORAGE_ENDPOINT`) can't tell apart from MinIO or R2.

## Decision

Use SeaweedFS's S3 gateway (`chrislusf/seaweedfs:latest`, `weed server
-s3 -s3.port=9000`) as the `seaweedfs` service in `docker-compose.yml`
for local dev, in place of MinIO. No application code changes: the
`ObjectStorage` interface and `S3Provider` implementation are
S3-API-generic by design (ADR-0001), so this is purely a
`docker-compose.yml`/environment-variable change.

The dev bucket is created by the backend itself at startup
(`S3Provider.EnsureBucket`, called from `Application.Bootstrap`) rather
than a separate `mc`-based init container — `minio/mc` hit the same
Docker Hub restriction, and self-provisioning removes the dependency on
any external client image entirely, local or not.

Production is unaffected: it targets Cloudflare R2, which was never
MinIO.

## Consequences

**Easier:** `docker compose up --build` works without a Docker Hub login
or account. One fewer moving part (no separate bucket-init container).

**Harder:** the architecture document's mention of MinIO for local dev
is now stale until it's updated to name SeaweedFS instead — read this
ADR as the current source of truth for that one detail in the meantime.
Unlike a login wall, this isn't expected to reverse itself — the
community images are gone by policy, not by a temporary restriction —
so this decision is not point-in-time in the way an outage or rate
limit would be.

**Explicitly not doing:**
- Switching production away from R2, or building a from-source MinIO
  Docker image to restore parity with the architecture document's
  literal wording. `S3Provider` already treats every S3-compatible
  backend identically (ADR-0001), so nothing in the code benefits from
  MinIO specifically over SeaweedFS — the only reason to still want
  MinIO here would be matching the architecture document's exact
  wording, which is cheaper to just update than to build a compile-time
  dependency on MinIO's own Go toolchain build for.
- Running MinIO natively on the host (`go install
  github.com/minio/minio@latest`) as part of this project's checked-in
  dev stack — it would work for one person's machine, but a
  host-run process isn't reachable from the `backend` container by
  service name the way `docker-compose.yml`'s other services are
  (it would need `host.docker.internal`, which isn't reliable across
  platforms), breaking the "`docker compose up --build` starts
  everything" guarantee the README makes. Running MinIO natively for
  personal, ad hoc exploration of its admin console is fine — it's
  just not this repo's dev stack.
