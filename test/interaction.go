package main

import (
	"net/http"
	"net/url"
	"strconv"
)

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

// testVideoLikeAction 点赞/取消点赞视频
// actionType: 1=点赞, 2=取消点赞
func testVideoLikeAction(client *http.Client, baseURL, token, videoID string, actionType int) Result[VideoLikeActionResponse] {
	body := map[string]any{
		"video_id":    videoID,
		"action_type": actionType,
	}
	var result VideoLikeActionResponse
	status, raw, err := doJSON(client, http.MethodPost, baseURL+"/api/v1/interaction/like", body, token, &result)
	return Result[VideoLikeActionResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

// testListLikedVideos 获取用户点赞的视频列表
func testListLikedVideos(client *http.Client, baseURL, userID string, pageNum, pageSize int) Result[ListLikedVideosResponse] {
	u, err := url.Parse(baseURL + "/api/v1/interaction/like/list")
	if err != nil {
		return Result[ListLikedVideosResponse]{Err: err}
	}
	q := u.Query()
	q.Set("user_id", userID)
	q.Set("page_num", strconv.Itoa(pageNum))
	q.Set("page_size", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	var result ListLikedVideosResponse
	status, raw, err := doRequest(client, http.MethodGet, u.String(), "", "", nil, &result)
	return Result[ListLikedVideosResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

// testPublishComment 发布评论
func testPublishComment(client *http.Client, baseURL, token, videoID, content string) Result[PublishCommentResponse] {
	body := map[string]any{
		"video_id": videoID,
		"content":  content,
	}
	var result PublishCommentResponse
	status, raw, err := doJSON(client, http.MethodPost, baseURL+"/api/v1/interaction/comment", body, token, &result)
	return Result[PublishCommentResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

// testListUserComments 获取用户发表的评论列表
func testListUserComments(client *http.Client, baseURL, userID string, pageNum, pageSize int) Result[ListUserCommentsResponse] {
	u, err := url.Parse(baseURL + "/api/v1/interaction/comment/list")
	if err != nil {
		return Result[ListUserCommentsResponse]{Err: err}
	}
	q := u.Query()
	q.Set("user_id", userID)
	q.Set("page_num", strconv.Itoa(pageNum))
	q.Set("page_size", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	var result ListUserCommentsResponse
	status, raw, err := doRequest(client, http.MethodGet, u.String(), "", "", nil, &result)
	return Result[ListUserCommentsResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}

// testDeleteComment 删除评论
func testDeleteComment(client *http.Client, baseURL, token, commentID string) Result[DeleteCommentResponse] {
	body := map[string]any{
		"comment_id": commentID,
	}
	var result DeleteCommentResponse
	status, raw, err := doJSON(client, http.MethodPost, baseURL+"/api/v1/interaction/comment/delete", body, token, &result)
	return Result[DeleteCommentResponse]{Data: result, StatusCode: status, RawBody: raw, Err: err}
}
