package middleware

import (
	"context"
	"os"
	"strings"
	"testing"

	"example.com/fanone/work5/pkg/auth"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.SetHandlers(app.HandlersChain{AuthMiddleware()})

	ctx.Next(context.Background())

	if ctx.Response.StatusCode() != 401 {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, "请提供认证令牌") {
		t.Fatalf("response body = %s", body)
	}
}

func TestAuthMiddlewareRejectsBadFormat(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Request.Header.Set("Authorization", "invalid-token")
	ctx.SetHandlers(app.HandlersChain{AuthMiddleware()})

	ctx.Next(context.Background())

	if ctx.Response.StatusCode() != 401 {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, "认证令牌格式错误") {
		t.Fatalf("response body = %s", body)
	}
}

func TestAuthMiddlewareAllowsValidAccessToken(t *testing.T) {
	resetJWTManagerForTest(t)

	jwtMgr := auth.GetJWTManager()
	accessToken, _, err := jwtMgr.GenerateTokenPair(12, "middleware-user")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	var nextCalled bool
	ctx := app.NewContext(0)
	ctx.Request.Header.Set("Authorization", "Bearer "+accessToken)
	ctx.SetHandlers(app.HandlersChain{
		AuthMiddleware(),
		func(_ context.Context, c *app.RequestContext) {
			nextCalled = true
			if c.GetUint("user_id") != 12 {
				t.Fatalf("user_id = %d, want 12", c.GetUint("user_id"))
			}
			if c.GetString("username") != "middleware-user" {
				t.Fatalf("username = %s, want middleware-user", c.GetString("username"))
			}
		},
	})

	ctx.Next(context.Background())

	if !nextCalled {
		t.Fatal("后续处理器未执行")
	}
}

func TestAuthMiddlewareRejectsRefreshToken(t *testing.T) {
	resetJWTManagerForTest(t)

	jwtMgr := auth.GetJWTManager()
	_, refreshToken, err := jwtMgr.GenerateTokenPair(15, "refresh-only")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	ctx := app.NewContext(0)
	ctx.Request.Header.Set("Authorization", "Bearer "+refreshToken)
	ctx.SetHandlers(app.HandlersChain{AuthMiddleware()})

	ctx.Next(context.Background())

	if ctx.Response.StatusCode() != 401 {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, "\"code\":1005") {
		t.Fatalf("response body = %s", body)
	}
}

func resetJWTManagerForTest(t *testing.T) {
	t.Helper()

	oldSecret := os.Getenv("JWT_SECRET")
	t.Cleanup(func() {
		if oldSecret == "" {
			_ = os.Unsetenv("JWT_SECRET")
		} else {
			_ = os.Setenv("JWT_SECRET", oldSecret)
		}
	})

	if err := os.Setenv("JWT_SECRET", "middleware-test-secret"); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
}
