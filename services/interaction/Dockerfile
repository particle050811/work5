FROM golang:1.25 AS builder

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.org

COPY go.mod go.sum ./
COPY gen ./gen
COPY pkg ./pkg
COPY idl ./idl
COPY services ./services

RUN cd /src/services/interaction && GOWORK=off CGO_ENABLED=0 go build -o /out/interaction-service .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/interaction-service /app/interaction-service

EXPOSE 9003

ENTRYPOINT ["/app/interaction-service"]
