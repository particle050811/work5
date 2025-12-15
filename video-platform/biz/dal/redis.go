package dal

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// initRedis 初始化 Redis 客户端
func initRedis() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Fatal("[Redis] REDIS_ADDR 未配置，服务启动失败")
	}

	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")
	db := 0
	if dbStr != "" {
		if parsed, err := strconv.Atoi(dbStr); err == nil {
			db = parsed
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("[Redis] 连接失败: %v", err)
	}

	log.Printf("[Redis] 连接成功: %s", addr)
	return client
}
