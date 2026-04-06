package main

import (
	"log"
	"net"
	"os"
	"strings"

	"example.com/fanone/gen-rpc/kitex_gen/user/v1/userservice"
	"example.com/fanone/services/user/internal/handler"
	"example.com/fanone/services/user/internal/repository"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	"github.com/joho/godotenv"
	etcd "github.com/kitex-contrib/registry-etcd"
)

const (
	userServiceName = "fanone.user"
	defaultUserAddr = "0.0.0.0:9001"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，将使用系统环境变量")
	}
	repository.Init()

	addr, err := net.ResolveTCPAddr("tcp", getEnv("USER_RPC_ADDR", defaultUserAddr))
	if err != nil {
		log.Fatalf("解析 user-service 地址失败: %v", err)
	}

	reg, err := etcd.NewEtcdRegistry(splitCSV(getEnv("ETCD_ENDPOINTS", "127.0.0.1:2379")))
	if err != nil {
		log.Fatalf("初始化 etcd 注册中心失败: %v", err)
	}

	svr := userservice.NewServer(
		handler.NewRPCHandler(repository.GetStore()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: userServiceName}),
		server.WithServiceAddr(addr),
		server.WithRegistry(reg),
	)

	log.Printf("user-service 启动中，监听 RPC 地址: %s", addr.String())
	if err := svr.Run(); err != nil {
		log.Fatalf("user-service 启动失败: %v", err)
	}
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
