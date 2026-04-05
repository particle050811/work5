package auth

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestJWTManagerGenerateAndValidateTokenPair(t *testing.T) {
	manager := &JWTManager{
		secretKey:          []byte("unit-test-secret"),
		accessTokenExpiry:  15 * time.Minute,
		refreshTokenExpiry: 7 * 24 * time.Hour,
	}

	accessToken, refreshToken, err := manager.GenerateTokenPair(42, "tester")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}
	if accessToken == "" || refreshToken == "" {
		t.Fatal("GenerateTokenPair() 返回了空 token")
	}

	accessClaims, err := manager.ValidateAccessToken(accessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if accessClaims.UserID != 42 || accessClaims.Username != "tester" || accessClaims.TokenType != AccessToken {
		t.Fatalf("access claims 不符合预期: %+v", accessClaims)
	}

	refreshClaims, err := manager.ValidateRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken() error = %v", err)
	}
	if refreshClaims.UserID != 42 || refreshClaims.Username != "tester" || refreshClaims.TokenType != RefreshToken {
		t.Fatalf("refresh claims 不符合预期: %+v", refreshClaims)
	}
}

func TestJWTManagerValidateAccessTokenRejectsRefreshToken(t *testing.T) {
	manager := &JWTManager{
		secretKey:          []byte("unit-test-secret"),
		accessTokenExpiry:  15 * time.Minute,
		refreshTokenExpiry: 7 * 24 * time.Hour,
	}

	_, refreshToken, err := manager.GenerateTokenPair(7, "refresh-user")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	_, err = manager.ValidateAccessToken(refreshToken)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("ValidateAccessToken() error = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestJWTManagerParseTokenErrors(t *testing.T) {
	manager := &JWTManager{
		secretKey:          []byte("unit-test-secret"),
		accessTokenExpiry:  15 * time.Minute,
		refreshTokenExpiry: 7 * 24 * time.Hour,
	}

	if _, err := manager.ParseToken("not-a-jwt"); !errors.Is(err, ErrTokenMalformed) {
		t.Fatalf("ParseToken() malformed error = %v, want %v", err, ErrTokenMalformed)
	}

	expiredToken, err := manager.generateToken(1, "expired-user", AccessToken, -time.Minute)
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}
	if _, err := manager.ParseToken(expiredToken); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("ParseToken() expired error = %v, want %v", err, ErrTokenExpired)
	}
}

func TestJWTManagerRefreshTokens(t *testing.T) {
	manager := &JWTManager{
		secretKey:          []byte("unit-test-secret"),
		accessTokenExpiry:  15 * time.Minute,
		refreshTokenExpiry: 7 * 24 * time.Hour,
	}

	_, refreshToken, err := manager.GenerateTokenPair(9, "refreshable")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	newAccessToken, newRefreshToken, err := manager.RefreshTokens(refreshToken)
	if err != nil {
		t.Fatalf("RefreshTokens() error = %v", err)
	}
	if newAccessToken == "" || newRefreshToken == "" {
		t.Fatal("RefreshTokens() 返回了空 token")
	}
	if _, err := manager.ValidateAccessToken(newAccessToken); err != nil {
		t.Fatalf("新 access token 校验失败: %v", err)
	}
	if _, err := manager.ValidateRefreshToken(newRefreshToken); err != nil {
		t.Fatalf("新 refresh token 校验失败: %v", err)
	}
}

func TestGetJWTManagerSingleton(t *testing.T) {
	oldSecret := os.Getenv("JWT_SECRET")
	t.Cleanup(func() {
		if oldSecret == "" {
			_ = os.Unsetenv("JWT_SECRET")
		} else {
			_ = os.Setenv("JWT_SECRET", oldSecret)
		}
		jwtManager = nil
	})

	if err := os.Setenv("JWT_SECRET", "singleton-secret"); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
	jwtManager = nil

	first := GetJWTManager()
	second := GetJWTManager()
	if first == nil || second == nil {
		t.Fatal("GetJWTManager() 返回了 nil")
	}
	if first != second {
		t.Fatal("GetJWTManager() 未返回单例实例")
	}
}
