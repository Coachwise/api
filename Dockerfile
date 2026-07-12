# syntax=docker/dockerfile:1

# ---- build stage: compile the three static binaries ----
FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app     ./cmd/app     \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker  ./cmd/worker  \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

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
EXPOSE 8000
CMD ["/app/bin/app"]
