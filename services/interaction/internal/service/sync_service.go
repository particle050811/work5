package service

import (
	"context"
	"fmt"

	"example.com/fanone/services/interaction/internal/repository/db"
	"example.com/fanone/services/interaction/internal/repository/model"
)

// SyncUser 同步用户副本到互动服务库。
func (s *InteractionService) SyncUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("用户数据不能为空")
	}
	if err := db.UpsertUser(s.store, user); err != nil {
		return fmt.Errorf("同步用户副本失败: %w", err)
	}
	return nil
}

// SyncVideo 同步视频副本到互动服务库。
func (s *InteractionService) SyncVideo(ctx context.Context, video *model.Video) error {
	if video == nil {
		return fmt.Errorf("视频数据不能为空")
	}
	if err := db.UpsertVideo(s.store, video); err != nil {
		return fmt.Errorf("同步视频副本失败: %w", err)
	}
	return nil
}

// ListVideoComments 获取某个视频的评论列表。
func (s *InteractionService) ListVideoComments(ctx context.Context, videoID uint, offset, limit int) ([]db.VideoCommentWithUser, int64, error) {
	return db.ListTopLevelCommentsByVideo(s.store, videoID, offset, limit)
}
