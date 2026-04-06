package db

import "gorm.io/gorm"

// DBProvider 提供数据库访问能力。
type DBProvider interface {
	DB() *gorm.DB
}
