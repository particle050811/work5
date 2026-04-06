package service

import (
	"context"
	"log"
	"time"

	"example.com/fanone/services/user/internal/repository/model"
)

const syncTimeout = 3 * time.Second

func (s *UserService) syncUserReplicasBestEffort(user *model.User, logPattern string) {
	if s.syncer == nil || user == nil {
		return
	}
	payload := &SyncUserPayload{
		ID:        user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
		defer cancel()
		if err := s.syncer.SyncUser(ctx, payload); err != nil {
			log.Printf(logPattern, user.ID, err)
		}
	}()
}
