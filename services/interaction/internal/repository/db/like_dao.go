package db

import (
	"errors"
	"example.com/fanone/services/interaction/internal/repository/model"

	"gorm.io/gorm"
)

// GetVideoLikeUnscoped 查询点赞记录（包含软删除记录）
func GetVideoLikeUnscoped(store DBProvider, userID, videoID uint) (*model.VideoLike, error) {
	var like model.VideoLike
	err := store.DB().Unscoped().Where("user_id = ? AND video_id = ?", userID, videoID).First(&like).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &like, nil
}

// CreateVideoLike 创建点赞记录
func CreateVideoLike(store DBProvider, like *model.VideoLike) error {
	return store.DB().Create(like).Error
}

// RestoreVideoLike 恢复软删除的点赞记录
func RestoreVideoLike(store DBProvider, id uint) error {
	return store.DB().Unscoped().Model(&model.VideoLike{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

// SoftDeleteVideoLike 软删除点赞记录
func SoftDeleteVideoLike(store DBProvider, id uint) error {
	return store.DB().Delete(&model.VideoLike{}, id).Error
}

// ListVideoLikesByUser 获取用户点赞的视频 ID 列表（分页）
func ListVideoLikesByUser(store DBProvider, userID uint, offset, limit int) ([]uint, int64, error) {
	var total int64
	base := store.DB().Model(&model.VideoLike{}).Where("user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var likes []model.VideoLike
	if err := base.Order("created_at desc").Offset(offset).Limit(limit).Find(&likes).Error; err != nil {
		return nil, 0, err
	}

	videoIDs := make([]uint, 0, len(likes))
	for _, l := range likes {
		videoIDs = append(videoIDs, l.VideoID)
	}
	return videoIDs, total, nil
}

// IncreaseVideoLikeCount 更新视频点赞数
func IncreaseVideoLikeCount(store DBProvider, videoID uint, delta int64) error {
	return store.DB().Model(&model.Video{}).Where("id = ?", videoID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}
