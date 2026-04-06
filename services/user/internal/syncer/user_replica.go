package syncer

import (
	"context"

	interactionv1 "example.com/fanone/gen-rpc/kitex_gen/interaction/v1"
	interactionclient "example.com/fanone/gen-rpc/kitex_gen/interaction/v1/interactionservice"
	videov1 "example.com/fanone/gen-rpc/kitex_gen/video/v1"
	videoclient "example.com/fanone/gen-rpc/kitex_gen/video/v1/videoservice"
	"example.com/fanone/services/user/internal/service"
)

type UserReplicaSyncer struct {
	video       videoclient.Client
	interaction interactionclient.Client
}

func NewUserReplicaSyncer(video videoclient.Client, interaction interactionclient.Client) *UserReplicaSyncer {
	if video == nil || interaction == nil {
		return nil
	}
	return &UserReplicaSyncer{video: video, interaction: interaction}
}

func (s *UserReplicaSyncer) SyncUser(ctx context.Context, user *service.SyncUserPayload) error {
	if s == nil || user == nil {
		return nil
	}
	if _, err := s.video.SyncUser(ctx, &videov1.SyncUserRequest{
		Id:        uint64(user.ID),
		Username:  user.Username,
		AvatarUrl: user.AvatarURL,
	}); err != nil {
		return err
	}
	if _, err := s.interaction.SyncUser(ctx, &interactionv1.SyncUserRequest{
		Id:        uint64(user.ID),
		Username:  user.Username,
		AvatarUrl: user.AvatarURL,
	}); err != nil {
		return err
	}
	return nil
}
