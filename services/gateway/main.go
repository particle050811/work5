package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	interactionv1 "example.com/fanone/gen-rpc/kitex_gen/interaction/v1"
	interactionclient "example.com/fanone/gen-rpc/kitex_gen/interaction/v1/interactionservice"
	userrpcv1 "example.com/fanone/gen-rpc/kitex_gen/user/v1"
	userclient "example.com/fanone/gen-rpc/kitex_gen/user/v1/userservice"
	videorpcv1 "example.com/fanone/gen-rpc/kitex_gen/video/v1"
	videoclient "example.com/fanone/gen-rpc/kitex_gen/video/v1/videoservice"
	"example.com/fanone/work5/docs/swagger"
	api "example.com/fanone/work5/idl/http/gen/v1"
	"example.com/fanone/work5/pkg/logger"
	hzmiddleware "example.com/fanone/work5/pkg/middleware"
	"example.com/fanone/work5/pkg/response"
	"example.com/fanone/work5/pkg/storage"
	"example.com/fanone/work5/pkg/util"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/hertz-contrib/cors"
	"github.com/joho/godotenv"
	etcd "github.com/kitex-contrib/registry-etcd"
)

const (
	defaultGatewayAddr = ":8888"
	userServiceName    = "fanone.user"
	videoServiceName   = "fanone.video"
	interactionName    = "fanone.interaction"
)

type gatewayClients struct {
	user        userclient.Client
	video       videoclient.Client
	interaction interactionclient.Client
}

func main() {
	if err := logger.Init(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("刷新日志缓冲失败: %v", err)
		}
	}()

	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，将使用系统环境变量")
	}

	clients, err := newGatewayClients()
	if err != nil {
		log.Fatalf("初始化 RPC 客户端失败: %v", err)
	}

	h := server.Default(server.WithHostPorts(getEnv("GATEWAY_HTTP_ADDR", defaultGatewayAddr)))
	h.Use(hzmiddleware.RequestLogMiddleware())
	h.Use(cors.Default())

	registerRoutes(h, clients)
	storage.BindStatic(h)
	swagger.BindSwagger(h)

	log.Printf("gateway 启动中，监听 HTTP 地址: %s", getEnv("GATEWAY_HTTP_ADDR", defaultGatewayAddr))
	h.Spin()
}

func registerRoutes(h *server.Hertz, clients *gatewayClients) {
	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "pong")
	})

	h.POST("/api/v1/user/register", clients.register)
	h.POST("/api/v1/user/login", clients.login)
	h.POST("/api/v1/user/refresh", clients.refreshToken)
	h.GET("/api/v1/user/info", clients.getUserInfo)

	auth := h.Group("/api/v1")
	auth.Use(hzmiddleware.AuthMiddleware())
	auth.POST("/user/avatar", clients.uploadAvatar)
	auth.POST("/video/publish", clients.publishVideo)
	auth.POST("/interaction/like", clients.videoLikeAction)
	auth.POST("/interaction/comment", clients.publishComment)
	auth.POST("/interaction/comment/delete", clients.deleteComment)
	auth.POST("/relation/action", clients.relationAction)
	auth.GET("/relation/friend/list", clients.listFriends)

	h.GET("/api/v1/video/list", clients.listPublishedVideos)
	h.GET("/api/v1/video/search", clients.searchVideos)
	h.GET("/api/v1/video/comments", clients.listVideoComments)
	h.GET("/api/v1/video/hot", clients.getHotVideos)
	h.GET("/api/v1/interaction/like/list", clients.listLikedVideos)
	h.GET("/api/v1/interaction/comment/list", clients.listUserComments)
	h.GET("/api/v1/relation/following/list", clients.listFollowings)
	h.GET("/api/v1/relation/follower/list", clients.listFollowers)
}

func newGatewayClients() (*gatewayClients, error) {
	endpoints := splitCSV(getEnv("ETCD_ENDPOINTS", "127.0.0.1:2379"))
	resolver, err := etcd.NewEtcdResolver(endpoints)
	if err != nil {
		return nil, err
	}

	userCli, err := userclient.NewClient(userServiceName, client.WithResolver(resolver))
	if err != nil {
		return nil, err
	}
	videoCli, err := videoclient.NewClient(videoServiceName, client.WithResolver(resolver))
	if err != nil {
		return nil, err
	}
	interactionCli, err := interactionclient.NewClient(interactionName, client.WithResolver(resolver))
	if err != nil {
		return nil, err
	}

	return &gatewayClients{
		user:        userCli,
		video:       videoCli,
		interaction: interactionCli,
	}, nil
}

func (g *gatewayClients) register(ctx context.Context, c *app.RequestContext) {
	var req api.RegisterRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &api.RegisterResponse{Base: response.ParamError(err.Error())})
		return
	}

	resp, err := g.user.Register(ctx, &userrpcv1.RegisterRequest{
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
	if err := g.syncUserReplicas(ctx, resp.GetUser()); err != nil {
		log.Printf("[用户模块][注册] 同步用户副本失败 username=%s: %v", req.Username, err)
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
	if err := g.syncUserReplicas(ctx, resp.GetUser()); err != nil {
		log.Printf("[用户模块][登录] 同步用户副本失败 username=%s: %v", req.Username, err)
	}
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
	if err := g.syncUserReplicas(ctx, resp.GetUser()); err != nil {
		log.Printf("[用户模块][获取用户信息] 同步用户副本失败 user_id=%d: %v", userID, err)
	}
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
	if err := g.syncUserReplicas(ctx, resp.GetUser()); err != nil {
		log.Printf("[用户模块][上传头像] 同步用户副本失败 user_id=%d: %v", userID, err)
	}
}

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

	resp, err := g.video.CreateVideo(ctx, &videorpcv1.CreateVideoRequest{
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
	if _, err := g.interaction.SyncVideo(ctx, &interactionv1.SyncVideoRequest{Video: interactionVideoFromVideoRPC(resp.GetVideo())}); err != nil {
		log.Printf("[视频模块][投稿] 同步互动视频副本失败 user_id=%d: %v", userID, err)
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
		Data: &api.VideoCommentList{
			Items: items,
			Total: resp.GetData().GetTotal(),
		},
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

	resp, err := g.interaction.VideoLikeAction(ctx, &interactionv1.VideoLikeActionRequest{
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
	if resp.GetAppliedDelta() != 0 {
		if _, syncErr := g.video.SyncVideoCounters(ctx, &videorpcv1.SyncVideoCountersRequest{
			VideoId:   uint64(videoID),
			LikeDelta: resp.GetAppliedDelta(),
		}); syncErr != nil {
			log.Printf("[互动模块][点赞操作] 同步视频计数失败 video_id=%d user_id=%d: %v", videoID, userID, syncErr)
			c.JSON(consts.StatusInternalServerError, &api.VideoLikeActionResponse{Base: response.InternalError()})
			return
		}
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
	if _, syncErr := g.video.SyncVideoCounters(ctx, &videorpcv1.SyncVideoCountersRequest{
		VideoId:      uint64(videoID),
		CommentDelta: 1,
	}); syncErr != nil {
		log.Printf("[互动模块][发布评论] 同步视频计数失败 video_id=%d user_id=%d: %v", videoID, userID, syncErr)
		c.JSON(consts.StatusInternalServerError, &api.PublishCommentResponse{Base: response.InternalError()})
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
		Data: &api.CommentListWithTotal{
			Items: items,
			Total: resp.GetData().GetTotal(),
		},
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

	resp, err := g.interaction.DeleteComment(ctx, &interactionv1.DeleteCommentRequest{
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
	if _, syncErr := g.video.SyncVideoCounters(ctx, &videorpcv1.SyncVideoCountersRequest{
		VideoId:      resp.GetVideoId(),
		CommentDelta: -1,
	}); syncErr != nil {
		log.Printf("[互动模块][删除评论] 同步视频计数失败 comment_id=%d user_id=%d: %v", commentID, userID, syncErr)
		c.JSON(consts.StatusInternalServerError, &api.DeleteCommentResponse{Base: response.InternalError()})
		return
	}

	c.JSON(consts.StatusOK, &api.DeleteCommentResponse{Base: response.Success("删除成功")})
}

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

func rpcUserToHTTP(user *userrpcv1.UserProfile) *api.User {
	if user == nil {
		return nil
	}
	return &api.User{
		Id:        fmt.Sprintf("%d", user.GetId()),
		Username:  user.GetUsername(),
		AvatarUrl: user.GetAvatarUrl(),
		CreatedAt: user.GetCreatedAt(),
		UpdatedAt: user.GetUpdatedAt(),
		DeletedAt: user.GetDeletedAt(),
	}
}

func rpcVideoListToHTTP(data *videorpcv1.VideoList) *api.VideoListWithTotal {
	if data == nil {
		return &api.VideoListWithTotal{}
	}
	items := make([]*api.Video, 0, len(data.GetItems()))
	for _, item := range data.GetItems() {
		items = append(items, &api.Video{
			Id:           fmt.Sprintf("%d", item.GetId()),
			UserId:       fmt.Sprintf("%d", item.GetUserId()),
			VideoUrl:     item.GetVideoUrl(),
			CoverUrl:     item.GetCoverUrl(),
			Title:        item.GetTitle(),
			Description:  item.GetDescription(),
			VisitCount:   item.GetVisitCount(),
			LikeCount:    item.GetLikeCount(),
			CommentCount: item.GetCommentCount(),
			CreatedAt:    item.GetCreatedAt(),
			UpdatedAt:    item.GetUpdatedAt(),
			DeletedAt:    item.GetDeletedAt(),
		})
	}
	return &api.VideoListWithTotal{Items: items, Total: data.GetTotal()}
}

func interactionVideoListToHTTP(data *interactionv1.VideoList) *api.VideoListWithTotal {
	if data == nil {
		return &api.VideoListWithTotal{}
	}
	items := make([]*api.Video, 0, len(data.GetItems()))
	for _, item := range data.GetItems() {
		items = append(items, &api.Video{
			Id:           fmt.Sprintf("%d", item.GetId()),
			UserId:       fmt.Sprintf("%d", item.GetUserId()),
			VideoUrl:     item.GetVideoUrl(),
			CoverUrl:     item.GetCoverUrl(),
			Title:        item.GetTitle(),
			Description:  item.GetDescription(),
			VisitCount:   item.GetVisitCount(),
			LikeCount:    item.GetLikeCount(),
			CommentCount: item.GetCommentCount(),
			CreatedAt:    item.GetCreatedAt(),
			UpdatedAt:    item.GetUpdatedAt(),
			DeletedAt:    item.GetDeletedAt(),
		})
	}
	return &api.VideoListWithTotal{Items: items, Total: data.GetTotal()}
}

func interactionVideoFromVideoRPC(video *videorpcv1.Video) *interactionv1.Video {
	if video == nil {
		return nil
	}
	return &interactionv1.Video{
		Id:           video.GetId(),
		UserId:       video.GetUserId(),
		VideoUrl:     video.GetVideoUrl(),
		CoverUrl:     video.GetCoverUrl(),
		Title:        video.GetTitle(),
		Description:  video.GetDescription(),
		VisitCount:   video.GetVisitCount(),
		LikeCount:    video.GetLikeCount(),
		CommentCount: video.GetCommentCount(),
		CreatedAt:    video.GetCreatedAt(),
		UpdatedAt:    video.GetUpdatedAt(),
		DeletedAt:    video.GetDeletedAt(),
	}
}

func (g *gatewayClients) syncUserReplicas(ctx context.Context, user *userrpcv1.UserProfile) error {
	if user == nil {
		return nil
	}
	if _, err := g.video.SyncUser(ctx, &videorpcv1.SyncUserRequest{
		Id:        user.GetId(),
		Username:  user.GetUsername(),
		AvatarUrl: user.GetAvatarUrl(),
	}); err != nil {
		return err
	}
	if _, err := g.interaction.SyncUser(ctx, &interactionv1.SyncUserRequest{
		Id:        user.GetId(),
		Username:  user.GetUsername(),
		AvatarUrl: user.GetAvatarUrl(),
	}); err != nil {
		return err
	}
	return nil
}

func socialListToHTTP(data *interactionv1.SocialList) *api.SocialListWithTotal {
	if data == nil {
		return &api.SocialListWithTotal{}
	}
	items := make([]*api.SocialProfile, 0, len(data.GetItems()))
	for _, item := range data.GetItems() {
		items = append(items, &api.SocialProfile{
			Id:        fmt.Sprintf("%d", item.GetId()),
			Username:  item.GetUsername(),
			AvatarUrl: item.GetAvatarUrl(),
		})
	}
	return &api.SocialListWithTotal{Items: items, Total: data.GetTotal()}
}

func idToString(id uint64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("%d", id)
}

func isRemoteErr(err error, target string) bool {
	if err == nil {
		return false
	}
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		for _, e := range joined.Unwrap() {
			if strings.Contains(e.Error(), target) {
				return true
			}
		}
	}
	return strings.Contains(err.Error(), target)
}

func getEnv(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
