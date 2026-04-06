package service

import (
	"context"
	"log"
	"time"
)

const syncTimeout = 3 * time.Second

func (s *InteractionService) syncVideoCountersBestEffort(videoID uint, likeDelta, commentDelta int64, logPattern string) {
	if s.videoSyncer == nil || videoID == 0 || (likeDelta == 0 && commentDelta == 0) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
		defer cancel()
		if err := s.videoSyncer.SyncVideoCounters(ctx, videoID, likeDelta, commentDelta); err != nil {
			log.Printf(logPattern, videoID, err)
		}
	}()
}
