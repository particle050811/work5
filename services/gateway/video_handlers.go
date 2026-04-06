package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	interactionv1 "example.com/fanone/gen-rpc/kitex_gen/interaction/v1"
	videorpcv1 "example.com/fanone/gen-rpc/kitex_gen/video/v1"
	api "example.com/fanone/work5/idl/http/gen/v1"
	"example.com/fanone/work5/pkg/response"
	"example.com/fanone/work5/pkg/storage"
	"example.com/fanone/work5/pkg/util"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func (g *gatewayClients) publishVideo(ctx context.Context, c *app.RequestContext) {
	var req api.PublishVideoRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.PublishVideoResponse{Base: response.ParamError(err.Error())})
		return
	}
	userID := c.GetUint("user_id")
	fileHeader, err := c.FormFile("video")
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.PublishVideoResponse{Base: response.ParamError("请上传视频文件,字段名为 video")})
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		c.JSON(consts.StatusBadRequest, &api.PublishVideoResponse{Base: response.ParamError("title 不能为空")})
		return
	}

	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = ".mp4"
	}
	filename := fmt.Sprintf("video_%d_%d%s", userID, time.Now().UnixNano(), ext)
	savePath := filepath.Join(storage.VideoDir(), filename)
	if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
		log.Printf("[视频模块][投稿] 保存视频文件失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.PublishVideoResponse{Base: response.InternalError()})
		return
	}

	_, err = g.video.CreateVideo(ctx, &videorpcv1.CreateVideoRequest{
		UserId:      uint64(userID),
		VideoUrl:    storage.VideoURL(filename),
		Title:       title,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		log.Printf("[视频模块][投稿] 创建视频记录失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.PublishVideoResponse{Base: response.InternalError()})
		return
	}
	c.JSON(consts.StatusOK, &api.PublishVideoResponse{Base: response.Success("投稿成功")})
}

func (g *gatewayClients) listPublishedVideos(ctx context.Context, c *app.RequestContext) {
	var req api.ListPublishedVideosRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListPublishedVideosResponse{Base: response.ParamError(err.Error())})
		return
	}
	if strings.TrimSpace(req.UserId) == "" {
		c.JSON(consts.StatusBadRequest, &api.ListPublishedVideosResponse{Base: response.ParamError("user_id 不能为空")})
		return
	}
	userID, err := util.ParseUint(req.UserId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListPublishedVideosResponse{Base: response.ParamError("user_id 格式错误")})
		return
	}

	resp, err := g.video.ListPublishedVideos(ctx, &videorpcv1.ListPublishedVideosRequest{
		UserId:   uint64(userID),
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	})
	if err != nil {
		log.Printf("[视频模块][发布列表] 查询发布列表失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.ListPublishedVideosResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.ListPublishedVideosResponse{
		Base: response.Success(),
		Data: rpcVideoListToHTTP(resp.GetData()),
	})
}

func (g *gatewayClients) searchVideos(ctx context.Context, c *app.RequestContext) {
	var req api.SearchVideosRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.SearchVideosResponse{Base: response.ParamError(err.Error())})
		return
	}

	resp, err := g.video.SearchVideos(ctx, &videorpcv1.SearchVideosRequest{
		Keywords: req.Keywords,
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
		FromDate: req.FromDate,
		ToDate:   req.ToDate,
		Username: req.Username,
		SortBy:   req.SortBy,
	})
	if err != nil {
		log.Printf("[视频模块][搜索视频] 查询视频失败 keywords=%s username=%s: %v", req.Keywords, req.Username, err)
		c.JSON(consts.StatusInternalServerError, &api.SearchVideosResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.SearchVideosResponse{
		Base: response.Success(),
		Data: rpcVideoListToHTTP(resp.GetData()),
	})
}

func (g *gatewayClients) listVideoComments(ctx context.Context, c *app.RequestContext) {
	var req api.ListVideoCommentsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListVideoCommentsResponse{Base: response.ParamError(err.Error())})
		return
	}
	if strings.TrimSpace(req.VideoId) == "" {
		c.JSON(consts.StatusBadRequest, &api.ListVideoCommentsResponse{Base: response.ParamError("video_id 不能为空")})
		return
	}
	videoID, err := util.ParseUint(req.VideoId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListVideoCommentsResponse{Base: response.ParamError("video_id 格式错误")})
		return
	}

	resp, err := g.interaction.ListVideoComments(ctx, &interactionv1.ListVideoCommentsRequest{
		VideoId:  uint64(videoID),
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	})
	if err != nil {
		log.Printf("[视频模块][视频评论列表] 查询评论列表失败 video_id=%d: %v", videoID, err)
		c.JSON(consts.StatusInternalServerError, &api.ListVideoCommentsResponse{Base: response.InternalError()})
		return
	}

	items := make([]*api.VideoComment, 0, len(resp.GetData().GetItems()))
	for _, item := range resp.GetData().GetItems() {
		items = append(items, &api.VideoComment{
			Id:        fmt.Sprintf("%d", item.GetId()),
			UserId:    fmt.Sprintf("%d", item.GetUserId()),
			Username:  item.GetUsername(),
			AvatarUrl: item.GetAvatarUrl(),
			Content:   item.GetContent(),
			LikeCount: item.GetLikeCount(),
			CreatedAt: item.GetCreatedAt(),
		})
	}

	c.JSON(consts.StatusOK, &api.ListVideoCommentsResponse{
		Base: response.Success(),
		Data: &api.VideoCommentList{Items: items, Total: resp.GetData().GetTotal()},
	})
}

func (g *gatewayClients) getHotVideos(ctx context.Context, c *app.RequestContext) {
	var req api.GetHotVideosRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.GetHotVideosResponse{Base: response.ParamError(err.Error())})
		return
	}

	resp, err := g.video.GetHotVideos(ctx, &videorpcv1.GetHotVideosRequest{
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	})
	if err != nil {
		log.Printf("[视频模块][热门排行榜] 查询热榜失败 page_num=%d page_size=%d: %v", req.PageNum, req.PageSize, err)
		c.JSON(consts.StatusInternalServerError, &api.GetHotVideosResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.GetHotVideosResponse{
		Base: response.Success(),
		Data: rpcVideoListToHTTP(resp.GetData()),
	})
}
