package model

import (
	"time"

	"gorm.io/gorm"
)

// VideoLike 视频点赞记录
type VideoLike struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"uniqueIndex:idx_user_video;not null" json:"user_id"`
	VideoID   uint           `gorm:"uniqueIndex:idx_user_video;index;not null" json:"video_id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // 软删除支持取消点赞后再次点赞
}

func (VideoLike) TableName() string {
	return "video_likes"
}
