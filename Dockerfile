# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Dependencies first: this layer stays cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
ARG VERSION=dev
# The templates are embedded, so the resulting binary is self-contained.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/scaffold .

# Runtime stage: a few MB instead of the ~1 GB of the golang image.
FROM alpine:3.20
LABEL org.opencontainers.image.title="Go-DDD-Scaffold" \
      org.opencontainers.image.source="https://github.com/Allan-Nava/Go-DDD-Scaffold" \
      maintainer="allan.nava@hiway.media"

COPY --from=builder /out/scaffold /usr/local/bin/scaffold

# Generate into a bind mount: docker run --rm -v "$PWD:/work" <image> init
WORKDIR /work
ENTRYPOINT ["scaffold"]
CMD ["--help"]
