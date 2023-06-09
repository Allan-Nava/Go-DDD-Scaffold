# syntax=docker/dockerfile:1.2
FROM golang:1.19-bullseye as builder
LABEL maintainer="allan.nava@hiway.media"
ENV DOCKER_BUILDKIT=1
#
WORKDIR /app
COPY go.mod ./
COPY . .
#
RUN --mount=type=cache,target=/go/pkg/mod \
   --mount=type=cache,target=/root/.cache/go-build go mod tidy
#ARG VERSION
RUN --mount=type=cache,target=/go/pkg/mod \
   --mount=type=cache,target=/root/.cache/go-build \
   CGO_ENABLED=0 go build -installsuffix cgo -ldflags "-X main.version=1" -o main .
#
#
ENTRYPOINT ["./main"]
CMD ["-h"]
#
