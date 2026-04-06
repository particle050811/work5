package syncer

import (
	"context"

	interactionv1 "example.com/fanone/gen-rpc/kitex_gen/interaction/v1"
	interactionclient "example.com/fanone/gen-rpc/kitex_gen/interaction/v1/interactionservice"
	"example.com/fanone/services/video/internal/service"
)

type InteractionVideoSyncer struct {
	client interactionclient.Client
}

func NewInteractionVideoSyncer(cli interactionclient.Client) *InteractionVideoSyncer {
	if cli == nil {
		return nil
	}
	return &InteractionVideoSyncer{client: cli}
}

func (s *InteractionVideoSyncer) SyncVideo(ctx context.Context, video *service.SyncVideoPayload) error {
	if s == nil || s.client == nil || video == nil {
		return nil
	}
	_, err := s.client.SyncVideo(ctx, &interactionv1.SyncVideoRequest{
		Video: &interactionv1.Video{
			Id:           uint64(video.ID),
			UserId:       uint64(video.UserID),
			VideoUrl:     video.VideoURL,
			CoverUrl:     video.CoverURL,
			Title:        video.Title,
			Description:  video.Description,
			VisitCount:   video.VisitCount,
			LikeCount:    video.LikeCount,
			CommentCount: video.CommentCount,
			CreatedAt:    video.CreatedAt,
			UpdatedAt:    video.UpdatedAt,
			DeletedAt:    video.DeletedAt,
		},
	})
	return err
}
