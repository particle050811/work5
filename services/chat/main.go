package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"

	chatv1 "example.com/fanone/gen-rpc/kitex_gen/chat/v1"
	"example.com/fanone/gen-rpc/kitex_gen/chat/v1/chatservice"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kserver "github.com/cloudwego/kitex/server"
	hzws "github.com/hertz-contrib/websocket"
	"github.com/joho/godotenv"
	etcd "github.com/kitex-contrib/registry-etcd"
)

const (
	chatServiceName = "fanone.chat"
	defaultChatAddr = "0.0.0.0:9004"
	defaultChatHTTP = ":8889"
)

type chatRPCHandler struct{}

func (h *chatRPCHandler) Ping(ctx context.Context, req *chatv1.PingRequest) (*chatv1.PingResponse, error) {
	message := strings.TrimSpace(req.GetMessage())
	if message == "" {
		message = "pong"
	}
	return &chatv1.PingResponse{Message: message}, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，将使用系统环境变量")
	}

	go runWebsocketServer()
	runRPCServer()
}

func runRPCServer() {
	addr, err := net.ResolveTCPAddr("tcp", getEnv("CHAT_RPC_ADDR", defaultChatAddr))
	if err != nil {
		log.Fatalf("解析 chat-service RPC 地址失败: %v", err)
	}

	reg, err := etcd.NewEtcdRegistry(splitCSV(getEnv("ETCD_ENDPOINTS", "127.0.0.1:2379")))
	if err != nil {
		log.Fatalf("初始化 etcd 注册中心失败: %v", err)
	}

	svr := chatservice.NewServer(
		&chatRPCHandler{},
		kserver.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: chatServiceName}),
		kserver.WithServiceAddr(addr),
		kserver.WithRegistry(reg),
	)

	log.Printf("chat-service 启动中，监听 RPC 地址: %s", addr.String())
	if err := svr.Run(); err != nil {
		log.Fatalf("chat-service RPC 启动失败: %v", err)
	}
}

func runWebsocketServer() {
	upgrader := hzws.HertzUpgrader{
		CheckOrigin: func(ctx *app.RequestContext) bool { return true },
	}

	h := server.Default(server.WithHostPorts(getEnv("CHAT_HTTP_ADDR", defaultChatHTTP)))
	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, map[string]string{"message": "pong"})
	})
	h.GET("/ws/chat", func(ctx context.Context, c *app.RequestContext) {
		if err := upgrader.Upgrade(c, func(conn *hzws.Conn) {
			defer conn.Close()

			for {
				mt, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				reply := append([]byte("echo: "), msg...)
				if err := conn.WriteMessage(mt, reply); err != nil {
					return
				}
			}
		}); err != nil {
			c.String(consts.StatusBadRequest, "upgrade websocket failed: %v", err)
		}
	})

	log.Printf("chat-service 启动中，监听 WebSocket/HTTP 地址: %s", getEnv("CHAT_HTTP_ADDR", defaultChatHTTP))
	h.Spin()
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
