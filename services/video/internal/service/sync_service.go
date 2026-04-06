package service

import (
	"context"
	"fmt"

	"example.com/fanone/services/video/internal/repository/db"
	"example.com/fanone/services/video/internal/repository/model"
)

// SyncUser 同步用户副本到视频服务库。
func (s *VideoService) SyncUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("用户数据不能为空")
	}
	if err := db.UpsertUser(s.store, user); err != nil {
		return fmt.Errorf("同步用户副本失败: %w", err)
	}
	return nil
}

// SyncVideoCounters 同步互动侧产生的视频热度计数变化。
func (s *VideoService) SyncVideoCounters(ctx context.Context, videoID uint, visitDelta, likeDelta, commentDelta int64) error {
	if videoID == 0 {
		return fmt.Errorf("video_id 不能为空")
	}
	if err := db.UpdateVideoCounters(s.store, videoID, visitDelta, likeDelta, commentDelta); err != nil {
		return fmt.Errorf("同步视频计数失败: %w", err)
	}
	return nil
}
