FROM golang:1.25-alpine AS builder

ARG SERVICE=paper-api

ENV SERVER_DIR=/papercompare
WORKDIR ${SERVER_DIR}

ENV GOPROXY=https://goproxy.cn,direct
ENV GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /papercompare/app ./cmd/${SERVICE}

FROM alpine:latest

WORKDIR /app

COPY --from=builder /papercompare/app /app/app
COPY --from=builder /papercompare/config /config

ENV CONFIG_PATH=/config

CMD ["./app"]
