package db

import (
	"time"
	"video-platform/biz/dal/model"

	"gorm.io/gorm"
)

type VideoCommentWithUser struct {
	ID        uint
	UserID    uint
	Username  string
	AvatarURL string
	Content   string
	LikeCount int64
	CreatedAt time.Time
}

// ListTopLevelCommentsByVideo 分页获取视频一级评论（附带评论者信息）
func ListTopLevelCommentsByVideo(store DBProvider, videoID uint, offset, limit int) ([]VideoCommentWithUser, int64, error) {
	base := store.DB().
		Table("comments").
		Select("comments.id, comments.user_id, users.username, users.avatar_url, comments.content, comments.like_count, comments.created_at").
		Joins("JOIN users ON users.id = comments.user_id").
		Where("comments.video_id = ?", videoID).
		Where("comments.parent_id IS NULL").
		Where("comments.deleted_at IS NULL")

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []VideoCommentWithUser
	if err := base.Order("comments.created_at desc").Offset(offset).Limit(limit).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// IncreaseVideoCommentCount 自增视频评论数
func IncreaseVideoCommentCount(store DBProvider, videoID uint, delta int64) error {
	return store.DB().Model(&model.Video{}).Where("id = ?", videoID).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}
