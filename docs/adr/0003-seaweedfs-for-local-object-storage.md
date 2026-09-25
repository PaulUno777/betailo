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
block. This matches MinIO's 2025 shift to requiring an account/login for
their container images.

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
Anyone reproducing this project's Docker Hub pull failure on their own
machine, at a later date, should re-verify it before assuming SeaweedFS
is still necessary — this is a point-in-time workaround for an external
policy, not a technical preference for SeaweedFS over MinIO on its own
merits.

**Explicitly not doing:** switching production away from R2, or adding
a MinIO Docker Hub login/credential to CI or local setup instructions —
both would treat a local-dev-only obstacle as if it also constrained
production, which it doesn't.
