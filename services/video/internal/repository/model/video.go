package model

import (
	"time"

	"gorm.io/gorm"
)

// Video 视频模型
type Video struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`
	VideoURL     string         `gorm:"size:500;not null" json:"video_url"`
	CoverURL     string         `gorm:"size:500" json:"cover_url"`
	Title        string         `gorm:"size:200;not null" json:"title"`
	Description  string         `gorm:"type:text" json:"description"`
	VisitCount   int64          `gorm:"not null;default:0" json:"visit_count"`
	LikeCount    int64          `gorm:"not null;default:0" json:"like_count"`
	CommentCount int64          `gorm:"not null;default:0" json:"comment_count"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Video) TableName() string {
	return "videos"
}
