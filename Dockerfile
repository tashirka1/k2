FROM ghcr.io/a-h/templ:latest AS templ
COPY --chown=65532:65532 . /app
WORKDIR /app
RUN ["templ", "generate"]

FROM golang:1.26-alpine AS builder
WORKDIR /app
RUN apk add --no-cache build-base sqlite-dev
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY --from=templ /app /app
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 GOOS=linux go build -tags fts5 -ldflags="-s -w" -o bin/k2 cmd/k2/main.go

FROM alpine:3.23 AS run
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /app/bin/k2 /app/bin/k2
# if you enable two lines below, container doesn't have access to docker.sock
# RUN adduser --disabled-password --gecos "" noroot && \
#     chown -R noroot:noroot /app
# USER noroot:noroot
EXPOSE 8000
CMD ["/app/bin/k2"]
