FROM golang:1.25.3-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/fanone-video .

RUN mkdir -p /out/storage/avatars /out/storage/videos

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S -G app app

COPY --from=builder --chown=app:app /out/fanone-video /app/fanone-video
COPY --from=builder --chown=app:app /out/storage /app/storage

USER app

EXPOSE 8888

ENTRYPOINT ["/app/fanone-video"]
