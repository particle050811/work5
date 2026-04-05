package service

import (
	"context"
	"errors"
	"os"
	"testing"

	"video-platform/biz/dal/model"
	"video-platform/pkg/auth"
)

func TestUserServiceRegisterAndLogin(t *testing.T) {
	resetJWTEnv(t)

	store, gdb := newTestStore(t)
	svc := NewUserService(store)

	registerResult, err := svc.Register(context.Background(), "service_user", "Password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registerResult.User == nil || registerResult.User.ID == 0 {
		t.Fatalf("Register() 返回用户异常: %+v", registerResult)
	}
	if registerResult.User.Password == "Password123" {
		t.Fatal("注册后密码未加密")
	}

	var storedUser model.User
	if err := gdb.First(&storedUser, registerResult.User.ID).Error; err != nil {
		t.Fatalf("查询注册用户失败: %v", err)
	}
	if !auth.CheckPassword("Password123", storedUser.Password) {
		t.Fatal("数据库中的密码哈希校验失败")
	}

	if _, err := svc.Register(context.Background(), "service_user", "Password123"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("重复注册错误 = %v, want %v", err, ErrUserExists)
	}

	loginResult, err := svc.Login(context.Background(), "service_user", "Password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if loginResult.AccessToken == "" || loginResult.RefreshToken == "" {
		t.Fatal("登录未生成 token 对")
	}
	if _, err := auth.GetJWTManager().ValidateAccessToken(loginResult.AccessToken); err != nil {
		t.Fatalf("access token 校验失败: %v", err)
	}
}

func TestUserServiceLoginErrorsAndRefreshToken(t *testing.T) {
	resetJWTEnv(t)

	store, gdb := newTestStore(t)
	svc := NewUserService(store)
	hashedPassword, err := auth.HashPassword("Password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := gdb.Create(&model.User{Username: "login_user", Password: hashedPassword}).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	if _, err := svc.Login(context.Background(), "missing_user", "Password123"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("用户不存在错误 = %v, want %v", err, ErrUserNotFound)
	}
	if _, err := svc.Login(context.Background(), "login_user", "bad-password"); !errors.Is(err, ErrPasswordWrong) {
		t.Fatalf("密码错误 = %v, want %v", err, ErrPasswordWrong)
	}

	loginResult, err := svc.Login(context.Background(), "login_user", "Password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	refreshResult, err := svc.RefreshToken(context.Background(), loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if refreshResult.AccessToken == "" || refreshResult.RefreshToken == "" {
		t.Fatal("刷新 token 返回为空")
	}

	if _, err := svc.RefreshToken(context.Background(), "bad-refresh-token"); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("无效 refresh token 错误 = %v, want %v", err, ErrTokenInvalid)
	}
}

func resetJWTEnv(t *testing.T) {
	t.Helper()

	oldSecret := os.Getenv("JWT_SECRET")
	t.Cleanup(func() {
		if oldSecret == "" {
			_ = os.Unsetenv("JWT_SECRET")
		} else {
			_ = os.Setenv("JWT_SECRET", oldSecret)
		}
	})

	if err := os.Setenv("JWT_SECRET", "service-test-secret"); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
}
