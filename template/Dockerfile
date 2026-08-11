# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Dependencies first: this layer stays cached until go.mod/go.sum change.
# "go.*" tolerates a project that has not run `go mod tidy` yet.
COPY go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/{{.ProjectName}} ./cmd

# Runtime stage: ~15 MB instead of the ~1 GB of the golang image.
FROM alpine:3.20
LABEL org.opencontainers.image.title="{{.ProjectName}}"

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app

WORKDIR /app
COPY --from=builder /out/{{.ProjectName}} /app/{{.ProjectName}}
COPY --from=builder /src/config /app/config

USER app
EXPOSE 8080

# The dotenv files are deliberately not copied: in a container the
# configuration comes from the environment.
ENTRYPOINT ["/app/{{.ProjectName}}"]
