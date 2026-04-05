package auth

import "testing"

func TestHashPasswordAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("fanone-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() 返回空哈希")
	}
	if hash == "fanone-password" {
		t.Fatal("HashPassword() 未对密码进行哈希")
	}

	if !CheckPassword("fanone-password", hash) {
		t.Fatal("CheckPassword() 未通过正确密码")
	}
	if CheckPassword("wrong-password", hash) {
		t.Fatal("CheckPassword() 错误地通过了错误密码")
	}
}
