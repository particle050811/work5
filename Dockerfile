FROM golang:1.25 AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY gen ./gen
COPY pkg ./pkg
COPY idl ./idl
COPY docs/swagger ./docs/swagger
COPY services ./services

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/services/gateway && GOWORK=off CGO_ENABLED=0 go build -o /out/gateway .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/gateway /app/gateway

EXPOSE 8888

ENTRYPOINT ["/app/gateway"]
