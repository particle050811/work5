package middleware

import (
	"context"
	"regexp"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestRequestLogMiddlewareUsesExistingRequestID(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Request.SetRequestURI("/ping")
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.Header.Set(requestIDHeader, "req-123")

	var seenRequestID string
	ctx.SetHandlers(app.HandlersChain{
		RequestLogMiddleware(),
		func(_ context.Context, c *app.RequestContext) {
			seenRequestID = c.GetString(requestIDKey)
			c.Set("user_id", uint(7))
			c.SetStatusCode(204)
		},
	})

	ctx.Next(context.Background())

	if seenRequestID != "req-123" {
		t.Fatalf("request_id = %s, want req-123", seenRequestID)
	}
	if string(ctx.Response.Header.Peek(requestIDHeader)) != "req-123" {
		t.Fatalf("response header request_id = %s", string(ctx.Response.Header.Peek(requestIDHeader)))
	}
}

func TestRequestLogMiddlewareGeneratesRequestID(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Request.SetRequestURI("/health")
	ctx.Request.Header.SetMethod("GET")
	ctx.SetHandlers(app.HandlersChain{
		RequestLogMiddleware(),
		func(_ context.Context, c *app.RequestContext) {
			c.SetStatusCode(200)
		},
	})

	ctx.Next(context.Background())

	requestID := ctx.GetString(requestIDKey)
	if requestID == "" {
		t.Fatal("未生成 request_id")
	}
	if matched := regexp.MustCompile("^[a-f0-9]{32}$").MatchString(requestID); !matched {
		t.Fatalf("request_id = %s, 格式不正确", requestID)
	}
	if string(ctx.Response.Header.Peek(requestIDHeader)) != requestID {
		t.Fatalf("response header request_id = %s, want %s", string(ctx.Response.Header.Peek(requestIDHeader)), requestID)
	}
}
