package main

import (
	"context"
	"log"
	"strings"

	interactionv1 "example.com/fanone/gen-rpc/kitex_gen/interaction/v1"
	api "example.com/fanone/work5/idl/http/gen/v1"
	"example.com/fanone/work5/pkg/response"
	"example.com/fanone/work5/pkg/util"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func (g *gatewayClients) relationAction(ctx context.Context, c *app.RequestContext) {
	var req api.RelationActionRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.RelationActionResponse{Base: response.ParamError(err.Error())})
		return
	}
	userID := c.GetUint("user_id")
	if strings.TrimSpace(req.ToUserId) == "" {
		c.JSON(consts.StatusBadRequest, &api.RelationActionResponse{Base: response.ParamError("to_user_id 不能为空")})
		return
	}
	targetUserID, err := util.ParseUint(req.ToUserId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.RelationActionResponse{Base: response.ParamError("to_user_id 格式错误")})
		return
	}
	if req.ActionType != 1 && req.ActionType != 2 {
		c.JSON(consts.StatusBadRequest, &api.RelationActionResponse{Base: response.ParamError("action_type 必须为1（关注）或2（取关）")})
		return
	}

	_, err = g.interaction.RelationAction(ctx, &interactionv1.RelationActionRequest{
		UserId:     uint64(userID),
		ToUserId:   uint64(targetUserID),
		ActionType: req.ActionType,
	})
	if err != nil {
		switch {
		case isRemoteErr(err, "不能关注自己"):
			c.JSON(consts.StatusBadRequest, &api.RelationActionResponse{Base: response.ParamError("不能关注自己")})
		case isRemoteErr(err, "用户不存在"):
			c.JSON(consts.StatusNotFound, &api.RelationActionResponse{Base: response.NotFound("用户不存在")})
		default:
			log.Printf("[社交模块][关注操作] 执行关注操作失败 user_id=%d target_user_id=%d: %v", userID, targetUserID, err)
			c.JSON(consts.StatusInternalServerError, &api.RelationActionResponse{Base: response.InternalError()})
		}
		return
	}

	msg := "关注成功"
	if req.ActionType == 2 {
		msg = "取消关注成功"
	}
	c.JSON(consts.StatusOK, &api.RelationActionResponse{Base: response.Success(msg)})
}

func (g *gatewayClients) listFollowings(ctx context.Context, c *app.RequestContext) {
	var req api.ListFollowingsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListFollowingsResponse{Base: response.ParamError(err.Error())})
		return
	}
	if strings.TrimSpace(req.UserId) == "" {
		c.JSON(consts.StatusBadRequest, &api.ListFollowingsResponse{Base: response.ParamError("user_id 不能为空")})
		return
	}
	userID, err := util.ParseUint(req.UserId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListFollowingsResponse{Base: response.ParamError("user_id 格式错误")})
		return
	}

	resp, err := g.interaction.ListFollowings(ctx, &interactionv1.ListFollowingsRequest{
		UserId:   uint64(userID),
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	})
	if err != nil {
		log.Printf("[社交模块][关注列表] 查询关注列表失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.ListFollowingsResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.ListFollowingsResponse{
		Base: response.Success(),
		Data: socialListToHTTP(resp.GetData()),
	})
}

func (g *gatewayClients) listFollowers(ctx context.Context, c *app.RequestContext) {
	var req api.ListFollowersRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListFollowersResponse{Base: response.ParamError(err.Error())})
		return
	}
	if strings.TrimSpace(req.UserId) == "" {
		c.JSON(consts.StatusBadRequest, &api.ListFollowersResponse{Base: response.ParamError("user_id 不能为空")})
		return
	}
	userID, err := util.ParseUint(req.UserId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListFollowersResponse{Base: response.ParamError("user_id 格式错误")})
		return
	}

	resp, err := g.interaction.ListFollowers(ctx, &interactionv1.ListFollowersRequest{
		UserId:   uint64(userID),
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	})
	if err != nil {
		log.Printf("[社交模块][粉丝列表] 查询粉丝列表失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.ListFollowersResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.ListFollowersResponse{
		Base: response.Success(),
		Data: socialListToHTTP(resp.GetData()),
	})
}

func (g *gatewayClients) listFriends(ctx context.Context, c *app.RequestContext) {
	var req api.ListFriendsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.ListFriendsResponse{Base: response.ParamError(err.Error())})
		return
	}
	userID := c.GetUint("user_id")

	resp, err := g.interaction.ListFriends(ctx, &interactionv1.ListFriendsRequest{
		UserId:   uint64(userID),
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	})
	if err != nil {
		log.Printf("[社交模块][好友列表] 查询好友列表失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.ListFriendsResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.ListFriendsResponse{
		Base: response.Success(),
		Data: socialListToHTTP(resp.GetData()),
	})
}
