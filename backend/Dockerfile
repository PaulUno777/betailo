# Multi-stage build: compile with the full Go toolchain, ship a tiny
# static binary in a distroless-style final image. No migrations/schemas
# directory needs to be copied into the final image — the $jsonSchema
# validators are embedded into the binary itself (see backend/schemas).

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
