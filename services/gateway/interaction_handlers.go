package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	interactionv1 "example.com/fanone/gen-rpc/kitex_gen/interaction/v1"
	api "example.com/fanone/work5/idl/http/gen/v1"
	"example.com/fanone/work5/pkg/response"
	"example.com/fanone/work5/pkg/util"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func (g *gatewayClients) videoLikeAction(ctx context.Context, c *app.RequestContext) {
	var req api.VideoLikeActionRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.VideoLikeActionResponse{Base: response.ParamError(err.Error())})
		return
	}
	userID := c.GetUint("user_id")
	if strings.TrimSpace(req.VideoId) == "" {
		c.JSON(consts.StatusBadRequest, &api.VideoLikeActionResponse{Base: response.ParamError("video_id 不能为空")})
		return
	}
	videoID, err := util.ParseUint(req.VideoId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.VideoLikeActionResponse{Base: response.ParamError("video_id 格式错误")})
		return
	}
	if req.ActionType != 1 && req.ActionType != 2 {
		c.JSON(consts.StatusBadRequest, &api.VideoLikeActionResponse{Base: response.ParamError("action_type 必须为 1（点赞）或 2（取消点赞）")})
		return
	}

	_, err = g.interaction.VideoLikeAction(ctx, &interactionv1.VideoLikeActionRequest{
		UserId:     uint64(userID),
		VideoId:    uint64(videoID),
		ActionType: req.ActionType,
	})
	if err != nil {
		if isRemoteErr(err, "视频不存在") {
			c.JSON(consts.StatusNotFound, &api.VideoLikeActionResponse{Base: response.NotFound("视频不存在")})
			return
		}
		log.Printf("[互动模块][点赞操作] 执行点赞操作失败 video_id=%d user_id=%d: %v", videoID, userID, err)
		c.JSON(consts.StatusInternalServerError, &api.VideoLikeActionResponse{Base: response.InternalError()})
		return
	}
	msg := "点赞成功"
	if req.ActionType == 2 {
		msg = "取消点赞成功"
	}
	c.JSON(consts.StatusOK, &api.VideoLikeActionResponse{Base: response.Success(msg)})
}

func (g *gatewayClients) listLikedVideos(ctx context.Context, c *app.RequestContext) {
	var req api.ListLikedVideosRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListLikedVideosResponse{Base: response.ParamError(err.Error())})
		return
	}
	if strings.TrimSpace(req.UserId) == "" {
		c.JSON(consts.StatusBadRequest, &api.ListLikedVideosResponse{Base: response.ParamError("user_id 不能为空")})
		return
	}
	userID, err := util.ParseUint(req.UserId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListLikedVideosResponse{Base: response.ParamError("user_id 格式错误")})
		return
	}

	resp, err := g.interaction.ListLikedVideos(ctx, &interactionv1.ListLikedVideosRequest{
		UserId:   uint64(userID),
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	})
	if err != nil {
		log.Printf("[互动模块][点赞列表] 查询点赞列表失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.ListLikedVideosResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.ListLikedVideosResponse{
		Base: response.Success(),
		Data: interactionVideoListToHTTP(resp.GetData()),
	})
}

func (g *gatewayClients) publishComment(ctx context.Context, c *app.RequestContext) {
	var req api.PublishCommentRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.PublishCommentResponse{Base: response.ParamError(err.Error())})
		return
	}
	userID := c.GetUint("user_id")
	if strings.TrimSpace(req.VideoId) == "" {
		c.JSON(consts.StatusBadRequest, &api.PublishCommentResponse{Base: response.ParamError("video_id 不能为空")})
		return
	}
	videoID, err := util.ParseUint(req.VideoId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.PublishCommentResponse{Base: response.ParamError("video_id 格式错误")})
		return
	}

	_, err = g.interaction.PublishComment(ctx, &interactionv1.PublishCommentRequest{
		UserId:  uint64(userID),
		VideoId: uint64(videoID),
		Content: req.Content,
	})
	if err != nil {
		switch {
		case isRemoteErr(err, "评论内容不能为空"):
			c.JSON(consts.StatusBadRequest, &api.PublishCommentResponse{Base: response.ParamError("评论内容不能为空")})
		case isRemoteErr(err, "评论内容过长"):
			c.JSON(consts.StatusBadRequest, &api.PublishCommentResponse{Base: response.ParamError("评论内容不能超过 1000 字符")})
		case isRemoteErr(err, "视频不存在"):
			c.JSON(consts.StatusNotFound, &api.PublishCommentResponse{Base: response.NotFound("视频不存在")})
		default:
			log.Printf("[互动模块][发布评论] 发布评论失败 video_id=%d user_id=%d: %v", videoID, userID, err)
			c.JSON(consts.StatusInternalServerError, &api.PublishCommentResponse{Base: response.InternalError()})
		}
		return
	}
	c.JSON(consts.StatusOK, &api.PublishCommentResponse{Base: response.Success("评论成功")})
}

func (g *gatewayClients) listUserComments(ctx context.Context, c *app.RequestContext) {
	var req api.ListUserCommentsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListUserCommentsResponse{Base: response.ParamError(err.Error())})
		return
	}
	if strings.TrimSpace(req.UserId) == "" {
		c.JSON(consts.StatusBadRequest, &api.ListUserCommentsResponse{Base: response.ParamError("user_id 不能为空")})
		return
	}
	userID, err := util.ParseUint(req.UserId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListUserCommentsResponse{Base: response.ParamError("user_id 格式错误")})
		return
	}

	resp, err := g.interaction.ListUserComments(ctx, &interactionv1.ListUserCommentsRequest{
		UserId:   uint64(userID),
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	})
	if err != nil {
		log.Printf("[互动模块][评论列表] 查询评论列表失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.ListUserCommentsResponse{Base: response.InternalError()})
		return
	}

	items := make([]*api.Comment, 0, len(resp.GetData().GetItems()))
	for _, item := range resp.GetData().GetItems() {
		items = append(items, &api.Comment{
			Id:         fmt.Sprintf("%d", item.GetId()),
			UserId:     fmt.Sprintf("%d", item.GetUserId()),
			VideoId:    fmt.Sprintf("%d", item.GetVideoId()),
			ParentId:   idToString(item.GetParentId()),
			LikeCount:  item.GetLikeCount(),
			ChildCount: item.GetChildCount(),
			Content:    item.GetContent(),
			CreatedAt:  item.GetCreatedAt(),
			UpdatedAt:  item.GetUpdatedAt(),
			DeletedAt:  item.GetDeletedAt(),
		})
	}

	c.JSON(consts.StatusOK, &api.ListUserCommentsResponse{
		Base: response.Success(),
		Data: &api.CommentListWithTotal{Items: items, Total: resp.GetData().GetTotal()},
	})
}

func (g *gatewayClients) deleteComment(ctx context.Context, c *app.RequestContext) {
	var req api.DeleteCommentRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.DeleteCommentResponse{Base: response.ParamError(err.Error())})
		return
	}
	userID := c.GetUint("user_id")
	if strings.TrimSpace(req.CommentId) == "" {
		c.JSON(consts.StatusBadRequest, &api.DeleteCommentResponse{Base: response.ParamError("comment_id 不能为空")})
		return
	}
	commentID, err := util.ParseUint(req.CommentId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.DeleteCommentResponse{Base: response.ParamError("comment_id 格式错误")})
		return
	}

	_, err = g.interaction.DeleteComment(ctx, &interactionv1.DeleteCommentRequest{
		UserId:    uint64(userID),
		CommentId: uint64(commentID),
	})
	if err != nil {
		switch {
		case isRemoteErr(err, "评论不存在"):
			c.JSON(consts.StatusNotFound, &api.DeleteCommentResponse{Base: response.NotFound("评论不存在")})
		case isRemoteErr(err, "无权限操作"):
			c.JSON(consts.StatusForbidden, &api.DeleteCommentResponse{Base: response.Forbidden("无权删除他人评论")})
		default:
			log.Printf("[互动模块][删除评论] 删除评论失败 comment_id=%d user_id=%d: %v", commentID, userID, err)
			c.JSON(consts.StatusInternalServerError, &api.DeleteCommentResponse{Base: response.InternalError()})
		}
		return
	}
	c.JSON(consts.StatusOK, &api.DeleteCommentResponse{Base: response.Success("删除成功")})
}
