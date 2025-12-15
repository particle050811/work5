package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "http://localhost:8888"

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

type PublishVideoResponse struct {
	Base BaseResponse `json:"base"`
}

type Video struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	VideoURL     string `json:"video_url"`
	CoverURL     string `json:"cover_url"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	VisitCount   int64  `json:"visit_count"`
	LikeCount    int64  `json:"like_count"`
	CommentCount int64  `json:"comment_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	DeletedAt    string `json:"deleted_at"`
}

type VideoListWithTotal struct {
	Items []Video `json:"items"`
	Total int64   `json:"total"`
}

type ListPublishedVideosResponse struct {
	Base BaseResponse       `json:"base"`
	Data VideoListWithTotal `json:"data"`
}

type SearchVideosResponse struct {
	Base BaseResponse       `json:"base"`
	Data VideoListWithTotal `json:"data"`
}

type GetHotVideosResponse struct {
	Base BaseResponse       `json:"base"`
	Data VideoListWithTotal `json:"data"`
}

type VideoComment struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Content   string `json:"content"`
	LikeCount int64  `json:"like_count"`
	CreatedAt string `json:"created_at"`
}

type VideoCommentList struct {
	Items []VideoComment `json:"items"`
	Total int64          `json:"total"`
}

type ListVideoCommentsResponse struct {
	Base BaseResponse     `json:"base"`
	Data VideoCommentList `json:"data"`
}

// 请求结果封装，包含错误信息
type Result[T any] struct {
	Data       T
	StatusCode int
	RawBody    string
	Err        error
}

func main() {
	fmt.Println("========== FanOne 视频平台 API 测试 ==========")
	fmt.Println()

	baseURL := getBaseURL()
	client := &http.Client{Timeout: 10 * time.Second}

	// 先检查服务是否可用
	fmt.Println("【0】检查服务连接...")
	if !checkServerAvailable(client, baseURL) {
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
	registerResult := testRegister(client, baseURL, username, password)
	if registerResult.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", registerResult.Err)
	} else if registerResult.Data.Base.Code == 0 {
		fmt.Printf("    ✓ 注册成功: %s\n", registerResult.Data.Base.Msg)
	} else {
		fmt.Printf("    ✗ 注册失败: %s（HTTP %d）\n", registerResult.Data.Base.Msg, registerResult.StatusCode)
	}
	fmt.Println()

	// 2. 测试重复注册
	fmt.Println("【2】测试重复注册（应该失败）")
	registerResult2 := testRegister(client, baseURL, username, password)
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
	loginResult := testLogin(client, baseURL, username, password)
	if loginResult.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", loginResult.Err)
	} else if loginResult.Data.Base.Code == 0 {
		fmt.Printf("    ✓ 登录成功\n")
		fmt.Printf("    - 用户ID: %s\n", loginResult.Data.Data.ID)
		fmt.Printf("    - 用户名: %s\n", loginResult.Data.Data.Username)
		fmt.Printf("    - AccessToken: %s...\n", truncate(loginResult.Data.AccessToken, 50))
		fmt.Printf("    - RefreshToken: %s...\n", truncate(loginResult.Data.RefreshToken, 50))
	} else {
		fmt.Printf("    ✗ 登录失败: %s（HTTP %d）\n", loginResult.Data.Base.Msg, loginResult.StatusCode)
	}
	fmt.Println()

	// 4. 测试错误密码登录
	fmt.Println("【4】测试错误密码登录（应该失败）")
	loginResult2 := testLogin(client, baseURL, username, "wrongpassword")
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
		userInfoResult := testGetUserInfo(client, baseURL, loginResult.Data.Data.ID)
		if userInfoResult.Err != nil {
			fmt.Printf("    ✗ 请求失败: %v\n", userInfoResult.Err)
		} else if userInfoResult.Data.Base.Code == 0 {
			fmt.Printf("    ✓ 获取成功\n")
			fmt.Printf("    - 用户ID: %s\n", userInfoResult.Data.Data.ID)
			fmt.Printf("    - 用户名: %s\n", userInfoResult.Data.Data.Username)
			fmt.Printf("    - 创建时间: %s\n", userInfoResult.Data.Data.CreatedAt)
		} else {
			fmt.Printf("    ✗ 获取失败: %s（HTTP %d）\n", userInfoResult.Data.Base.Msg, userInfoResult.StatusCode)
		}
	} else {
		fmt.Println("    - 跳过（登录失败，无用户ID）")
	}
	fmt.Println()

	// 6. 测试刷新令牌
	fmt.Println("【6】测试刷新令牌")
	if loginResult.Err == nil && loginResult.Data.RefreshToken != "" {
		refreshResult := testRefreshToken(client, baseURL, loginResult.Data.RefreshToken)
		if refreshResult.Err != nil {
			fmt.Printf("    ✗ 请求失败: %v\n", refreshResult.Err)
		} else if refreshResult.Data.Base.Code == 0 {
			fmt.Printf("    ✓ 刷新成功\n")
			fmt.Printf("    - 新 AccessToken: %s...\n", truncate(refreshResult.Data.AccessToken, 50))
			fmt.Printf("    - 新 RefreshToken: %s...\n", truncate(refreshResult.Data.RefreshToken, 50))
		} else {
			fmt.Printf("    ✗ 刷新失败: %s（HTTP %d）\n", refreshResult.Data.Base.Msg, refreshResult.StatusCode)
		}
	} else {
		fmt.Println("    - 跳过（登录失败，无 RefreshToken）")
	}
	fmt.Println()

	// 7. 测试无效令牌刷新
	fmt.Println("【7】测试无效令牌刷新（应该失败）")
	refreshResult2 := testRefreshToken(client, baseURL, "invalid_token")
	if refreshResult2.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", refreshResult2.Err)
	} else if refreshResult2.Data.Base.Code != 0 {
		fmt.Printf("    ✓ 符合预期，无效令牌被拒绝: %s\n", refreshResult2.Data.Base.Msg)
	} else {
		fmt.Printf("    ✗ 不符合预期，无效令牌应该刷新失败\n")
	}
	fmt.Println()

	// ====== 视频模块 ======
	fmt.Println("========== 视频模块 ==========")

	accessToken := ""
	userID := ""
	if loginResult.Err == nil && loginResult.Data.Base.Code == 0 {
		accessToken = loginResult.Data.AccessToken
		userID = loginResult.Data.Data.ID
	}

	// 8. 测试投稿
	fmt.Println("【8】测试视频投稿（需要登录）")
	videoTitle := fmt.Sprintf("test video %d", time.Now().Unix())
	videoDesc := "这是一个用于自动化测试的投稿视频（内容为随机字节，不是真实视频）"
	videoFilePath, cleanup, err := prepareVideoFile()
	if err != nil {
		fmt.Printf("    ✗ 准备视频文件失败: %v\n", err)
	} else {
		defer cleanup()
		publishResult := testPublishVideo(client, baseURL, accessToken, videoTitle, videoDesc, videoFilePath)
		if publishResult.Err != nil {
			fmt.Printf("    ✗ 请求失败: %v\n", publishResult.Err)
		} else if publishResult.Data.Base.Code == 0 {
			fmt.Printf("    ✓ 投稿成功: %s\n", publishResult.Data.Base.Msg)
		} else {
			fmt.Printf("    ✗ 投稿失败: %s（HTTP %d）\n", publishResult.Data.Base.Msg, publishResult.StatusCode)
			if publishResult.RawBody != "" {
				fmt.Printf("    - 响应体: %s\n", truncate(publishResult.RawBody, 200))
			}
		}
	}
	fmt.Println()

	// 9. 测试发布列表
	fmt.Println("【9】测试发布列表")
	var latestVideo *Video
	if userID == "" {
		fmt.Println("    - 跳过（登录失败，无 user_id）")
	} else {
		listResult := testListPublishedVideos(client, baseURL, userID, 1, 10)
		if listResult.Err != nil {
			fmt.Printf("    ✗ 请求失败: %v\n", listResult.Err)
		} else if listResult.Data.Base.Code == 0 {
			fmt.Printf("    ✓ 获取成功，总数: %d，本页: %d\n", listResult.Data.Data.Total, len(listResult.Data.Data.Items))
			if len(listResult.Data.Data.Items) > 0 {
				latestVideo = &listResult.Data.Data.Items[0]
				fmt.Printf("    - 最新视频ID: %s\n", latestVideo.ID)
				fmt.Printf("    - 标题: %s\n", latestVideo.Title)
				fmt.Printf("    - 视频URL: %s\n", latestVideo.VideoURL)
			}
		} else {
			fmt.Printf("    ✗ 获取失败: %s（HTTP %d）\n", listResult.Data.Base.Msg, listResult.StatusCode)
		}
	}
	fmt.Println()

	// 10. 测试视频文件是否可访问
	fmt.Println("【10】测试视频文件可访问（静态资源）")
	if latestVideo == nil || latestVideo.VideoURL == "" {
		fmt.Println("    - 跳过（无可用 video_url）")
	} else {
		ok, status, err := checkStaticAvailable(client, baseURL, latestVideo.VideoURL)
		if err != nil {
			fmt.Printf("    ✗ 请求失败: %v\n", err)
		} else if ok {
			fmt.Printf("    ✓ 可访问（HTTP %d）\n", status)
		} else {
			fmt.Printf("    ✗ 不可访问（HTTP %d）\n", status)
		}
	}
	fmt.Println()

	// 11. 测试搜索视频
	fmt.Println("【11】测试搜索视频")
	searchResult := testSearchVideos(client, baseURL, videoTitle, 1, 10, "", 0, 0, "latest")
	if searchResult.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", searchResult.Err)
	} else if searchResult.Data.Base.Code == 0 {
		fmt.Printf("    ✓ 搜索成功，总数: %d，本页: %d\n", searchResult.Data.Data.Total, len(searchResult.Data.Data.Items))
		found := false
		for _, v := range searchResult.Data.Data.Items {
			if v.Title == videoTitle {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("    ✓ 符合预期：能搜到刚投稿的视频\n")
		} else {
			fmt.Printf("    ✗ 不符合预期：未搜到刚投稿的视频\n")
		}
	} else {
		fmt.Printf("    ✗ 搜索失败: %s（HTTP %d）\n", searchResult.Data.Base.Msg, searchResult.StatusCode)
	}
	fmt.Println()

	// 12. 测试热门排行榜（要求 Redis 缓存）
	fmt.Println("【12】测试热门排行榜")
	hotResult := testGetHotVideos(client, baseURL, 1, 10)
	if hotResult.Err != nil {
		fmt.Printf("    ✗ 请求失败: %v\n", hotResult.Err)
	} else if hotResult.Data.Base.Code == 0 {
		fmt.Printf("    ✓ 获取成功，总数: %d，本页: %d\n", hotResult.Data.Data.Total, len(hotResult.Data.Data.Items))
	} else {
		fmt.Printf("    ✗ 获取失败: %s（HTTP %d）\n", hotResult.Data.Base.Msg, hotResult.StatusCode)
	}
	fmt.Println()

	// 13. 测试视频评论列表（新视频应为空）
	fmt.Println("【13】测试视频评论列表（新视频应为空）")
	if latestVideo == nil || latestVideo.ID == "" {
		fmt.Println("    - 跳过（无 video_id）")
	} else {
		commentsResult := testListVideoComments(client, baseURL, latestVideo.ID, 1, 10)
		if commentsResult.Err != nil {
			fmt.Printf("    ✗ 请求失败: %v\n", commentsResult.Err)
		} else if commentsResult.Data.Base.Code == 0 {
			fmt.Printf("    ✓ 获取成功，总数: %d，本页: %d\n", commentsResult.Data.Data.Total, len(commentsResult.Data.Data.Items))
			if commentsResult.Data.Data.Total == 0 {
				fmt.Printf("    ✓ 符合预期：新视频评论为空\n")
			} else {
				fmt.Printf("    - 提示：该视频已有评论（可能是历史数据）\n")
			}
		} else {
			fmt.Printf("    ✗ 获取失败: %s（HTTP %d）\n", commentsResult.Data.Base.Msg, commentsResult.StatusCode)
		}
	}
	fmt.Println()

	fmt.Println("========== 测试完成 ==========")
}

func getBaseURL() string {
	v := strings.TrimSpace(os.Getenv("BASE_URL"))
	if v == "" {
		return defaultBaseURL
	}
	return strings.TrimRight(v, "/")
}

func checkServerAvailable(client *http.Client, baseURL string) bool {
	resp, err := client.Get(baseURL + "/ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true
}

func testRegister(client *http.Client, baseURL, username, password string) Result[RegisterResponse] {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	var result RegisterResponse
	status, raw, err := doJSON(client, http.MethodPost, baseURL+"/api/v1/user/register", body, "", &result)
	return Result[RegisterResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func testLogin(client *http.Client, baseURL, username, password string) Result[LoginResponse] {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	var result LoginResponse
	status, raw, err := doJSON(client, http.MethodPost, baseURL+"/api/v1/user/login", body, "", &result)
	return Result[LoginResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func testGetUserInfo(client *http.Client, baseURL, userID string) Result[GetUserInfoResponse] {
	var result GetUserInfoResponse
	u := baseURL + "/api/v1/user/info?user_id=" + url.QueryEscape(userID)
	status, raw, err := doRequest(client, http.MethodGet, u, "", "", nil, &result)
	return Result[GetUserInfoResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func testRefreshToken(client *http.Client, baseURL, refreshToken string) Result[RefreshTokenResponse] {
	body := map[string]string{
		"refresh_token": refreshToken,
	}
	var result RefreshTokenResponse
	status, raw, err := doJSON(client, http.MethodPost, baseURL+"/api/v1/user/refresh", body, "", &result)
	return Result[RefreshTokenResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func doJSON(client *http.Client, method, url string, body any, token string, out any) (int, string, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, "", err
		}
		reader = bytes.NewBuffer(b)
	}
	return doRequest(client, method, url, "application/json", token, reader, out)
}

func doRequest(client *http.Client, method, url, contentType, token string, body io.Reader, out any) (int, string, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, "", err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}
	raw := string(rawBytes)

	if out != nil && len(rawBytes) > 0 {
		if err := json.Unmarshal(rawBytes, out); err != nil {
			return resp.StatusCode, raw, fmt.Errorf("解析 JSON 失败: %w（响应体: %s）", err, truncate(raw, 200))
		}
	}
	return resp.StatusCode, raw, nil
}

func testPublishVideo(client *http.Client, baseURL, accessToken, title, description, filePath string) Result[PublishVideoResponse] {
	if strings.TrimSpace(accessToken) == "" {
		return Result[PublishVideoResponse]{Err: fmt.Errorf("缺少 access_token（请先登录）")}
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("title", title)
	_ = w.WriteField("description", description)

	f, err := os.Open(filePath)
	if err != nil {
		return Result[PublishVideoResponse]{Err: err}
	}
	defer f.Close()

	part, err := w.CreateFormFile("video", filepath.Base(filePath))
	if err != nil {
		return Result[PublishVideoResponse]{Err: err}
	}
	if _, err := io.Copy(part, f); err != nil {
		return Result[PublishVideoResponse]{Err: err}
	}
	if err := w.Close(); err != nil {
		return Result[PublishVideoResponse]{Err: err}
	}

	var result PublishVideoResponse
	status, raw, err := doRequest(client, http.MethodPost, baseURL+"/api/v1/video/publish", w.FormDataContentType(), accessToken, &buf, &result)
	return Result[PublishVideoResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func testListPublishedVideos(client *http.Client, baseURL, userID string, pageNum, pageSize int) Result[ListPublishedVideosResponse] {
	u, err := url.Parse(baseURL + "/api/v1/video/list")
	if err != nil {
		return Result[ListPublishedVideosResponse]{Err: err}
	}
	q := u.Query()
	q.Set("user_id", userID)
	q.Set("page_num", strconv.Itoa(pageNum))
	q.Set("page_size", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	var result ListPublishedVideosResponse
	status, raw, err := doRequest(client, http.MethodGet, u.String(), "", "", nil, &result)
	return Result[ListPublishedVideosResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func testSearchVideos(client *http.Client, baseURL, keywords string, pageNum, pageSize int, username string, fromDate, toDate int64, sortBy string) Result[SearchVideosResponse] {
	u, err := url.Parse(baseURL + "/api/v1/video/search")
	if err != nil {
		return Result[SearchVideosResponse]{Err: err}
	}
	q := u.Query()
	if strings.TrimSpace(keywords) != "" {
		q.Set("keywords", keywords)
	}
	q.Set("page_num", strconv.Itoa(pageNum))
	q.Set("page_size", strconv.Itoa(pageSize))
	if strings.TrimSpace(username) != "" {
		q.Set("username", username)
	}
	if fromDate > 0 {
		q.Set("from_date", strconv.FormatInt(fromDate, 10))
	}
	if toDate > 0 {
		q.Set("to_date", strconv.FormatInt(toDate, 10))
	}
	if strings.TrimSpace(sortBy) != "" {
		q.Set("sort_by", sortBy)
	}
	u.RawQuery = q.Encode()

	var result SearchVideosResponse
	status, raw, err := doRequest(client, http.MethodGet, u.String(), "", "", nil, &result)
	return Result[SearchVideosResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func testGetHotVideos(client *http.Client, baseURL string, pageNum, pageSize int) Result[GetHotVideosResponse] {
	u, err := url.Parse(baseURL + "/api/v1/video/hot")
	if err != nil {
		return Result[GetHotVideosResponse]{Err: err}
	}
	q := u.Query()
	q.Set("page_num", strconv.Itoa(pageNum))
	q.Set("page_size", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	var result GetHotVideosResponse
	status, raw, err := doRequest(client, http.MethodGet, u.String(), "", "", nil, &result)
	return Result[GetHotVideosResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func testListVideoComments(client *http.Client, baseURL, videoID string, pageNum, pageSize int) Result[ListVideoCommentsResponse] {
	u, err := url.Parse(baseURL + "/api/v1/video/comments")
	if err != nil {
		return Result[ListVideoCommentsResponse]{Err: err}
	}
	q := u.Query()
	q.Set("video_id", videoID)
	q.Set("page_num", strconv.Itoa(pageNum))
	q.Set("page_size", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	var result ListVideoCommentsResponse
	status, raw, err := doRequest(client, http.MethodGet, u.String(), "", "", nil, &result)
	return Result[ListVideoCommentsResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

func checkStaticAvailable(client *http.Client, baseURL, path string) (bool, int, error) {
	u := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return false, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 300, resp.StatusCode, nil
}

func prepareVideoFile() (string, func(), error) {
	if p := strings.TrimSpace(os.Getenv("VIDEO_FILE")); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", nil, fmt.Errorf("VIDEO_FILE 不存在: %w", err)
		}
		return p, func() {}, nil
	}

	f, err := os.CreateTemp("", "fanone_test_*.mp4")
	if err != nil {
		return "", nil, err
	}
	path := f.Name()

	// 写入一些随机字节，保证上传的文件非空
	payload := bytes.Repeat([]byte("fanone-test-video\n"), 1024)
	if _, err := f.Write(payload); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}

	return path, func() { _ = os.Remove(path) }, nil
}
