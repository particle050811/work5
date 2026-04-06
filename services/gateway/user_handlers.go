package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	userrpcv1 "example.com/fanone/gen-rpc/kitex_gen/user/v1"
	api "example.com/fanone/work5/idl/http/gen/v1"
	"example.com/fanone/work5/pkg/response"
	"example.com/fanone/work5/pkg/storage"
	"example.com/fanone/work5/pkg/util"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func (g *gatewayClients) register(ctx context.Context, c *app.RequestContext) {
	var req api.RegisterRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.RegisterResponse{Base: response.ParamError(err.Error())})
		return
	}

	_, err := g.user.Register(ctx, &userrpcv1.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		if isRemoteErr(err, "用户名已存在") {
			c.JSON(consts.StatusOK, &api.RegisterResponse{Base: response.Error(response.CodeUserExists)})
			return
		}
		log.Printf("[用户模块][注册] 创建用户失败 username=%s: %v", req.Username, err)
		c.JSON(consts.StatusInternalServerError, &api.RegisterResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.RegisterResponse{Base: response.Success("注册成功")})
}

func (g *gatewayClients) login(ctx context.Context, c *app.RequestContext) {
	var req api.LoginRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.LoginResponse{Base: response.ParamError(err.Error())})
		return
	}

	resp, err := g.user.Login(ctx, &userrpcv1.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		if isRemoteErr(err, "用户不存在") {
			c.JSON(consts.StatusOK, &api.LoginResponse{Base: response.Error(response.CodeUserNotFound)})
			return
		}
		if isRemoteErr(err, "密码错误") {
			c.JSON(consts.StatusOK, &api.LoginResponse{Base: response.Error(response.CodePasswordWrong)})
			return
		}
		log.Printf("[用户模块][登录] 用户登录失败 username=%s: %v", req.Username, err)
		c.JSON(consts.StatusInternalServerError, &api.LoginResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.LoginResponse{
		Base:         response.Success("登录成功"),
		Data:         rpcUserToHTTP(resp.GetUser()),
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

func (g *gatewayClients) refreshToken(ctx context.Context, c *app.RequestContext) {
	var req api.RefreshTokenRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.RefreshTokenResponse{Base: response.ParamError(err.Error())})
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		c.JSON(consts.StatusBadRequest, &api.RefreshTokenResponse{Base: response.ParamError("refresh_token 不能为空")})
		return
	}

	resp, err := g.user.RefreshToken(ctx, &userrpcv1.RefreshTokenRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		if isRemoteErr(err, "令牌已过期") {
			c.JSON(consts.StatusOK, &api.RefreshTokenResponse{Base: response.Error(response.CodeTokenExpired)})
			return
		}
		c.JSON(consts.StatusOK, &api.RefreshTokenResponse{Base: response.Error(response.CodeTokenInvalid)})
		return
	}

	c.JSON(consts.StatusOK, &api.RefreshTokenResponse{
		Base:         response.Success("刷新成功"),
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

func (g *gatewayClients) getUserInfo(ctx context.Context, c *app.RequestContext) {
	var req api.GetUserInfoRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.GetUserInfoResponse{Base: response.ParamError(err.Error())})
		return
	}
	if strings.TrimSpace(req.UserId) == "" {
		c.JSON(consts.StatusBadRequest, &api.GetUserInfoResponse{Base: response.ParamError("user_id 不能为空")})
		return
	}
	userID, err := util.ParseUint(req.UserId)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.GetUserInfoResponse{Base: response.ParamError("user_id 格式错误")})
		return
	}

	resp, err := g.user.GetUserInfo(ctx, &userrpcv1.GetUserInfoRequest{UserId: uint64(userID)})
	if err != nil {
		if isRemoteErr(err, "用户不存在") {
			c.JSON(consts.StatusOK, &api.GetUserInfoResponse{Base: response.Error(response.CodeUserNotFound)})
			return
		}
		log.Printf("[用户模块][获取用户信息] 查询用户失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.GetUserInfoResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.GetUserInfoResponse{
		Base: response.Success(),
		Data: rpcUserToHTTP(resp.GetUser()),
	})
}

func (g *gatewayClients) uploadAvatar(ctx context.Context, c *app.RequestContext) {
	userID := c.GetUint("user_id")
	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(consts.StatusBadRequest, &api.UploadAvatarResponse{Base: response.ParamError("请上传头像文件")})
		return
	}

	filename := fmt.Sprintf("avatar_%d_%s", userID, filepath.Base(fileHeader.Filename))
	savePath := filepath.Join(storage.AvatarDir(), filename)
	if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
		log.Printf("[用户模块][上传头像] 保存头像文件失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.UploadAvatarResponse{Base: response.InternalError()})
		return
	}

	resp, err := g.user.UpdateAvatar(ctx, &userrpcv1.UpdateAvatarRequest{
		UserId:    uint64(userID),
		AvatarUrl: storage.AvatarURL(filename),
	})
	if err != nil {
		log.Printf("[用户模块][上传头像] 更新头像失败 user_id=%d: %v", userID, err)
		c.JSON(consts.StatusInternalServerError, &api.UploadAvatarResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.UploadAvatarResponse{
		Base: response.Success("头像上传成功"),
		Data: rpcUserToHTTP(resp.GetUser()),
	})
}
