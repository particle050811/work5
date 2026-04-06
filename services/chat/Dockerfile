FROM golang:1.25 AS builder

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.org

COPY go.mod go.sum ./
COPY gen ./gen
COPY pkg ./pkg
COPY idl ./idl
COPY services ./services

RUN cd /src/services/chat && GOWORK=off CGO_ENABLED=0 go build -o /out/chat-service .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/chat-service /app/chat-service

EXPOSE 8889 9004

ENTRYPOINT ["/app/chat-service"]
