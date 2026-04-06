package main

import (
	"context"
	"log"

	interactionclient "example.com/fanone/gen-rpc/kitex_gen/interaction/v1/interactionservice"
	userclient "example.com/fanone/gen-rpc/kitex_gen/user/v1/userservice"
	videoclient "example.com/fanone/gen-rpc/kitex_gen/video/v1/videoservice"
	"example.com/fanone/work5/docs/swagger"
	"example.com/fanone/work5/pkg/logger"
	hzmiddleware "example.com/fanone/work5/pkg/middleware"
	"example.com/fanone/work5/pkg/storage"
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
