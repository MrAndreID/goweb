# syntax=docker/dockerfile:1

FROM --platform=${BUILDPLATFORM} golang:1.27-alpine3.24 AS builder

ENV CGO_ENABLED=0 \
    GOTOOLCHAIN=local \
    GOFLAGS=-mod=readonly

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    go mod download

ARG TARGETOS TARGETARCH

RUN --mount=type=bind,source=cmd,target=cmd \
    --mount=type=bind,source=internal,target=internal \
    --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,id=go-build-${TARGETOS}-${TARGETARCH} \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
        -trimpath \
        -buildvcs=false \
        -ldflags="-s -w -buildid=" \
        -o /out/engine \
        ./cmd/web

FROM alpine:3.24 AS runtime

ENV TZ=Asia/Jakarta

RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /app/storage/log

WORKDIR /app

COPY --link --chmod=444 .env.example ./.env

COPY --link --chmod=555 --from=builder /out/engine ./engine

EXPOSE 10101

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z -w 2 127.0.0.1 "${APP_PORT:-10101}" || exit 1

ENTRYPOINT ["/app/engine"]
