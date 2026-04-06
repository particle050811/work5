package repository

import (
	"example.com/fanone/services/interaction/internal/repository/db"
	"example.com/fanone/services/interaction/internal/repository/model"
	"gorm.io/gorm"
)

// Store 统一数据访问层，聚合 MySQL 和 Redis
type Store struct {
	db    *gorm.DB
	redis RedisClient
}

var defaultStore *Store

// Init 初始化 Store（在 main.go 中调用）
func Init() {
	mysqlDB := db.InitMySQL()
	if err := mysqlDB.AutoMigrate(
		&model.User{},
		&model.Video{},
		&model.VideoLike{},
		&model.Comment{},
		&model.Follow{},
	); err != nil {
		panic(err)
	}

	defaultStore = &Store{
		db:    mysqlDB,
		redis: initRedis(), // 失败返回 nil，不 panic
	}
}

// GetStore 获取全局 Store 实例
func GetStore() *Store {
	return defaultStore
}

// DB 获取数据库实例
func (s *Store) DB() *gorm.DB {
	return s.db
}

// Redis 获取 Redis 客户端
func (s *Store) Redis() RedisClient {
	return s.redis
}

// HasRedis 检查 Redis 是否可用
func (s *Store) HasRedis() bool {
	return s != nil && s.redis != nil
}

// WithTx 在事务中执行操作，返回带事务的 Store
func (s *Store) WithTx(fn func(txStore *Store) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txStore := &Store{db: tx, redis: s.redis}
		return fn(txStore)
	})
}
