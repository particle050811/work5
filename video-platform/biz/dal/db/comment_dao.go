package db

import (
	"errors"
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

// GetCommentByID 根据 ID 获取评论
func GetCommentByID(store DBProvider, id uint) (*model.Comment, error) {
	var comment model.Comment
	err := store.DB().First(&comment, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &comment, nil
}

// CreateComment 创建评论
func CreateComment(store DBProvider, comment *model.Comment) error {
	return store.DB().Create(comment).Error
}

// ListCommentsByUser 获取用户发表的评论列表（分页）
func ListCommentsByUser(store DBProvider, userID uint, offset, limit int) ([]model.Comment, int64, error) {
	var total int64
	base := store.DB().Model(&model.Comment{}).Where("user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var comments []model.Comment
	if err := base.Order("created_at desc").Offset(offset).Limit(limit).Find(&comments).Error; err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}
