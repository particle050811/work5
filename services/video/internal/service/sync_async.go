package service

import (
	"context"
	"log"
	"time"

	"example.com/fanone/services/video/internal/repository/model"
)

const syncTimeout = 3 * time.Second

func (s *VideoService) syncVideoReplicaBestEffort(video *model.Video) {
	if s.syncer == nil || video == nil {
		return
	}
	payload := &SyncVideoPayload{
		ID:           video.ID,
		UserID:       video.UserID,
		VideoURL:     video.VideoURL,
		CoverURL:     video.CoverURL,
		Title:        video.Title,
		Description:  video.Description,
		VisitCount:   video.VisitCount,
		LikeCount:    video.LikeCount,
		CommentCount: video.CommentCount,
		CreatedAt:    video.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    video.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if video.DeletedAt.Valid {
		payload.DeletedAt = video.DeletedAt.Time.Format("2006-01-02 15:04:05")
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
		defer cancel()
		if err := s.syncer.SyncVideo(ctx, payload); err != nil {
			log.Printf("[视频模块][投稿] 同步互动视频副本失败 video_id=%d user_id=%d: %v", video.ID, video.UserID, err)
		}
	}()
}
