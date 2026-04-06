package repository

import (
	"log"

	"example.com/fanone/services/user/internal/repository/db"
	"example.com/fanone/services/user/internal/repository/model"
	"gorm.io/gorm"
)

// Store 统一数据访问层，聚合 MySQL 和 Redis
type Store struct {
	db *gorm.DB
}

var defaultStore *Store

// Init 初始化 Store（在 main.go 中调用）
func Init() {
	defaultStore = &Store{
		db: db.InitMySQL(),
	}
	autoMigrate(defaultStore.db)
}

// GetStore 获取全局 Store 实例
func GetStore() *Store {
	return defaultStore
}

// DB 获取数据库实例
func (s *Store) DB() *gorm.DB {
	return s.db
}

// WithTx 在事务中执行操作，返回带事务的 Store
func (s *Store) WithTx(fn func(txStore *Store) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txStore := &Store{db: tx}
		return fn(txStore)
	})
}

// autoMigrate 自动迁移所有模型
func autoMigrate(gormDB *gorm.DB) {
	err := gormDB.AutoMigrate(
		&model.User{},
	)
	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Println("数据库迁移完成")
}
