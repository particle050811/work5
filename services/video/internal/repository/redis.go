package repository

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// initRedis 初始化 Redis 客户端
func initRedis() *redis.Client {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		log.Printf("[Redis] REDIS_ADDR 未配置，Redis 缓存降级为直连数据库")
		return nil
	}

	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")
	db := 0
	if dbStr != "" {
		if parsed, err := strconv.Atoi(dbStr); err == nil {
			db = parsed
		} else {
			log.Printf("[Redis] REDIS_DB=%q 非法，回退使用默认 DB 0", dbStr)
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
		log.Printf("[Redis] 连接失败，Redis 缓存降级为直连数据库 addr=%s: %v", addr, err)
		_ = client.Close()
		return nil
	}

	log.Printf("[Redis] 连接成功: %s", addr)
	return client
}
