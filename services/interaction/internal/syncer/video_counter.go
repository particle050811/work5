package syncer

import (
	"context"

	videov1 "example.com/fanone/gen-rpc/kitex_gen/video/v1"
	videoclient "example.com/fanone/gen-rpc/kitex_gen/video/v1/videoservice"
)

type VideoCounterSyncer struct {
	client videoclient.Client
}

func NewVideoCounterSyncer(cli videoclient.Client) *VideoCounterSyncer {
	if cli == nil {
		return nil
	}
	return &VideoCounterSyncer{client: cli}
}

func (s *VideoCounterSyncer) SyncVideoCounters(ctx context.Context, videoID uint, likeDelta, commentDelta int64) error {
	if s == nil || s.client == nil || videoID == 0 || (likeDelta == 0 && commentDelta == 0) {
		return nil
	}
	_, err := s.client.SyncVideoCounters(ctx, &videov1.SyncVideoCountersRequest{
		VideoId:      uint64(videoID),
		LikeDelta:    likeDelta,
		CommentDelta: commentDelta,
	})
	return err
}
