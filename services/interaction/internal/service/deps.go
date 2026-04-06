package service

import "context"

type VideoCounterSyncer interface {
	SyncVideoCounters(ctx context.Context, videoID uint, likeDelta, commentDelta int64) error
}

type noopVideoCounterSyncer struct{}

func (noopVideoCounterSyncer) SyncVideoCounters(context.Context, uint, int64, int64) error {
	return nil
}
