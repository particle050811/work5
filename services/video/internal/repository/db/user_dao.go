package db

import (
	"errors"

	"example.com/fanone/services/video/internal/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

// GetUserByID 根据 ID 获取用户。
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
