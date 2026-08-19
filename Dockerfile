FROM ghcr.io/a-h/templ:latest AS templ
COPY --chown=65532:65532 . /app
WORKDIR /app
RUN ["templ", "generate"]

FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY --from=templ /app /app
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/k2 cmd/k2/main.go

FROM alpine:3.23 AS run
ENV GOGC=50 GOMEMLIMIT=40MiB
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /app/bin/k2 /app/bin/k2
RUN adduser --disabled-password --gecos "" noroot && \
    chown -R noroot:noroot /app
USER noroot:noroot
EXPOSE 8000
CMD ["/app/bin/k2"]
