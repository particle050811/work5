package model

import (
	"time"

	"gorm.io/gorm"
)

// Comment 评论模型（本次作业仅需一级评论）
type Comment struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"index;not null" json:"user_id"`
	VideoID    uint           `gorm:"index;not null" json:"video_id"`
	ParentID   *uint          `gorm:"index" json:"parent_id,omitempty"` // nil 表示一级评论
	LikeCount  int64          `gorm:"not null;default:0" json:"like_count"`
	ChildCount int64          `gorm:"not null;default:0" json:"child_count"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Comment) TableName() string {
	return "comments"
}
