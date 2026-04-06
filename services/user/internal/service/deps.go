package service

import "context"

type UserReplicaSyncer interface {
	SyncUser(ctx context.Context, user *SyncUserPayload) error
}

type SyncUserPayload struct {
	ID        uint
	Username  string
	AvatarURL string
}

type noopUserReplicaSyncer struct{}

func (noopUserReplicaSyncer) SyncUser(context.Context, *SyncUserPayload) error {
	return nil
}
