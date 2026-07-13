# syntax=docker/dockerfile:1

# ---- build stage: compile the three static binaries ----
# Track the latest 1.25 patch: the standard library is where most CVEs land, and
# they're only fixed by a newer Go, never by a dependency bump.
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# VERSION comes from the git tag in CI and is what /health reports.
ARG VERSION=dev
ENV LDFLAGS="-s -w -X coachwise/src/app/views.Version=${VERSION}"
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="${LDFLAGS}" -o /out/app     ./cmd/app     \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="${LDFLAGS}" -o /out/worker  ./cmd/worker  \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="${LDFLAGS}" -o /out/migrate ./cmd/migrate

# ---- runtime stage: slim image with binaries + runtime assets ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/ /app/bin/
# The API loads SQL query files from src/sql at runtime (sqldir); migrate reads
# src/sql/migrations. openapi.yaml is served at /openapi.yaml. config.yml is NOT
# baked in — it's mounted at runtime (holds prod secrets, gitignored).
COPY src/sql /app/src/sql
COPY openapi.yaml /app/openapi.yaml
RUN mkdir -p /app/uploads && chown -R app:app /app
USER app
ENV GIN_MODE=release
EXPOSE 8000
CMD ["/app/bin/app"]
