package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "http://localhost:8888"

// 响应结构
type BaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type RegisterResponse struct {
	Base BaseResponse `json:"base"`
}

type LoginResponse struct {
	Base         BaseResponse `json:"base"`
	Data         User         `json:"data"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	Base         BaseResponse `json:"base"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

type GetUserInfoResponse struct {
	Base BaseResponse `json:"base"`
	Data User         `json:"data"`
}

// 请求结果封装，包含错误信息
type Result[T any] struct {
	Data T
	Err  error
}

func main() {
	fmt.Println("========== FanOne 视频平台 API 测试 ==========")
	fmt.Println()

	// 先检查服务是否可用
	fmt.Println("【0】检查服务连接...")
	if !checkServerAvailable() {
		fmt.Println("    ✗ 无法连接到服务器，请确保服务已启动: go run .")
		fmt.Println("    服务地址:", baseURL)
		os.Exit(1)
	}
	fmt.Println("    ✓ 服务连接正常")
	fmt.Println()

	// 生成唯一用户名避免冲突
	username := fmt.Sprintf("testuser_%d", time.Now().Unix())
	password := "123456"

	// 1. 测试注册
	fmt.Println("【1】测试用户注册")
	registerResult := testRegister(username, password)
	if registerResult.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", registerResult.Err)
	} else if registerResult.Data.Base.Code == 0 {
		fmt.Printf("    ✓ 注册成功: %s\n", registerResult.Data.Base.Msg)
	} else {
		fmt.Printf("    ✗ 注册失败: %s\n", registerResult.Data.Base.Msg)
	}
	fmt.Println()

	// 2. 测试重复注册
	fmt.Println("【2】测试重复注册（应该失败）")
	registerResult2 := testRegister(username, password)
	if registerResult2.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", registerResult2.Err)
	} else if registerResult2.Data.Base.Code != 0 {
		fmt.Printf("    ✓ 符合预期，重复注册被拒绝: %s\n", registerResult2.Data.Base.Msg)
	} else {
		fmt.Printf("    ✗ 不符合预期，重复注册应该失败\n")
	}
	fmt.Println()

	// 3. 测试登录
	fmt.Println("【3】测试用户登录")
	loginResult := testLogin(username, password)
	if loginResult.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", loginResult.Err)
	} else if loginResult.Data.Base.Code == 0 {
		fmt.Printf("    ✓ 登录成功\n")
		fmt.Printf("    - 用户ID: %s\n", loginResult.Data.Data.ID)
		fmt.Printf("    - 用户名: %s\n", loginResult.Data.Data.Username)
		fmt.Printf("    - AccessToken: %s...\n", truncate(loginResult.Data.AccessToken, 50))
		fmt.Printf("    - RefreshToken: %s...\n", truncate(loginResult.Data.RefreshToken, 50))
	} else {
		fmt.Printf("    ✗ 登录失败: %s\n", loginResult.Data.Base.Msg)
	}
	fmt.Println()

	// 4. 测试错误密码登录
	fmt.Println("【4】测试错误密码登录（应该失败）")
	loginResult2 := testLogin(username, "wrongpassword")
	if loginResult2.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", loginResult2.Err)
	} else if loginResult2.Data.Base.Code != 0 {
		fmt.Printf("    ✓ 符合预期，错误密码被拒绝: %s\n", loginResult2.Data.Base.Msg)
	} else {
		fmt.Printf("    ✗ 不符合预期，错误密码应该登录失败\n")
	}
	fmt.Println()

	// 5. 测试获取用户信息
	fmt.Println("【5】测试获取用户信息")
	if loginResult.Err == nil && loginResult.Data.Data.ID != "" {
		userInfoResult := testGetUserInfo(loginResult.Data.Data.ID)
		if userInfoResult.Err != nil {
			fmt.Printf("    ✗ 请求失败: %v\n", userInfoResult.Err)
		} else if userInfoResult.Data.Base.Code == 0 {
			fmt.Printf("    ✓ 获取成功\n")
			fmt.Printf("    - 用户ID: %s\n", userInfoResult.Data.Data.ID)
			fmt.Printf("    - 用户名: %s\n", userInfoResult.Data.Data.Username)
			fmt.Printf("    - 创建时间: %s\n", userInfoResult.Data.Data.CreatedAt)
		} else {
			fmt.Printf("    ✗ 获取失败: %s\n", userInfoResult.Data.Base.Msg)
		}
	} else {
		fmt.Println("    - 跳过（登录失败，无用户ID）")
	}
	fmt.Println()

	// 6. 测试刷新令牌
	fmt.Println("【6】测试刷新令牌")
	if loginResult.Err == nil && loginResult.Data.RefreshToken != "" {
		refreshResult := testRefreshToken(loginResult.Data.RefreshToken)
		if refreshResult.Err != nil {
			fmt.Printf("    ✗ 请求失败: %v\n", refreshResult.Err)
		} else if refreshResult.Data.Base.Code == 0 {
			fmt.Printf("    ✓ 刷新成功\n")
			fmt.Printf("    - 新 AccessToken: %s...\n", truncate(refreshResult.Data.AccessToken, 50))
			fmt.Printf("    - 新 RefreshToken: %s...\n", truncate(refreshResult.Data.RefreshToken, 50))
		} else {
			fmt.Printf("    ✗ 刷新失败: %s\n", refreshResult.Data.Base.Msg)
		}
	} else {
		fmt.Println("    - 跳过（登录失败，无 RefreshToken）")
	}
	fmt.Println()

	// 7. 测试无效令牌刷新
	fmt.Println("【7】测试无效令牌刷新（应该失败）")
	refreshResult2 := testRefreshToken("invalid_token")
	if refreshResult2.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", refreshResult2.Err)
	} else if refreshResult2.Data.Base.Code != 0 {
		fmt.Printf("    ✓ 符合预期，无效令牌被拒绝: %s\n", refreshResult2.Data.Base.Msg)
	} else {
		fmt.Printf("    ✗ 不符合预期，无效令牌应该刷新失败\n")
	}
	fmt.Println()

	fmt.Println("========== 测试完成 ==========")
}

func checkServerAvailable() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + "/ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true
}

func testRegister(username, password string) Result[RegisterResponse] {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(baseURL+"/api/v1/user/register", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return Result[RegisterResponse]{Err: err}
	}
	defer resp.Body.Close()

	var result RegisterResponse
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)
	return Result[RegisterResponse]{Data: result}
}

func testLogin(username, password string) Result[LoginResponse] {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(baseURL+"/api/v1/user/login", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return Result[LoginResponse]{Err: err}
	}
	defer resp.Body.Close()

	var result LoginResponse
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)
	return Result[LoginResponse]{Data: result}
}

func testGetUserInfo(userID string) Result[GetUserInfoResponse] {
	resp, err := http.Get(baseURL + "/api/v1/user/info?user_id=" + userID)
	if err != nil {
		return Result[GetUserInfoResponse]{Err: err}
	}
	defer resp.Body.Close()

	var result GetUserInfoResponse
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)
	return Result[GetUserInfoResponse]{Data: result}
}

func testRefreshToken(refreshToken string) Result[RefreshTokenResponse] {
	body := map[string]string{
		"refresh_token": refreshToken,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(baseURL+"/api/v1/user/refresh", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return Result[RefreshTokenResponse]{Err: err}
	}
	defer resp.Body.Close()

	var result RefreshTokenResponse
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)
	return Result[RefreshTokenResponse]{Data: result}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
