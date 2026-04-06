package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"example.com/fanone/work5/pkg/logger"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
)

const (
	requestIDKey    = "request_id"
	requestIDHeader = "X-Request-Id"
)

// RequestLogMiddleware 记录请求日志并注入 request_id。
func RequestLogMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()

		requestID := string(c.GetHeader(requestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}

		c.Set(requestIDKey, requestID)
		c.Response.Header.Set(requestIDHeader, requestID)

		c.Next(ctx)

		path := c.FullPath()
		if path == "" {
			path = string(c.Path())
		}

		logger.L().Info("request completed",
			zap.String("request_id", requestID),
			zap.String("method", string(c.Method())),
			zap.String("path", path),
			zap.Uint("user_id", c.GetUint("user_id")),
			zap.Int("status_code", c.Response.StatusCode()),
			zap.Int64("cost_ms", time.Since(start).Milliseconds()),
		)
	}
}

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf[:])
}
