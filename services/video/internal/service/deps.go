package service

import "context"

type InteractionVideoSyncer interface {
	SyncVideo(ctx context.Context, video *SyncVideoPayload) error
}

type SyncVideoPayload struct {
	ID           uint
	UserID       uint
	VideoURL     string
	CoverURL     string
	Title        string
	Description  string
	VisitCount   int64
	LikeCount    int64
	CommentCount int64
	CreatedAt    string
	UpdatedAt    string
	DeletedAt    string
}

type noopInteractionVideoSyncer struct{}

func (noopInteractionVideoSyncer) SyncVideo(context.Context, *SyncVideoPayload) error {
	return nil
}
