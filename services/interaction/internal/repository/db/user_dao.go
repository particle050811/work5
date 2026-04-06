package db

import (
	"errors"

	"example.com/fanone/services/interaction/internal/repository/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DBProvider 数据库访问接口（与 dal.DBProvider 一致，避免循环依赖）
type DBProvider interface {
	DB() *gorm.DB
}

// ==================== 用户相关数据库操作 ====================

// CreateUser 创建用户
func CreateUser(store DBProvider, user *model.User) error {
	return store.DB().Create(user).Error
}

// UpsertUser 按主键同步用户副本。
func UpsertUser(store DBProvider, user *model.User) error {
	return store.DB().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"username":   user.Username,
			"avatar_url": user.AvatarURL,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(user).Error
}

// GetUserByID 根据 ID 获取用户
func GetUserByID(store DBProvider, id uint) (*model.User, error) {
	var user model.User
	err := store.DB().First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func GetUserByUsername(store DBProvider, username string) (*model.User, error) {
	var user model.User
	err := store.DB().Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func UpdateUser(store DBProvider, user *model.User) error {
	return store.DB().Save(user).Error
}

// UpdateUserAvatar 更新用户头像
func UpdateUserAvatar(store DBProvider, userID uint, avatarURL string) error {
	return store.DB().Model(&model.User{}).Where("id = ?", userID).Update("avatar_url", avatarURL).Error
}

// UserExists 检查用户名是否存在
func UserExists(store DBProvider, username string) (bool, error) {
	var count int64
	err := store.DB().Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
