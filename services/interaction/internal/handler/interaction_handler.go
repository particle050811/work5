package handler

import (
	"context"

	interactionv1 "example.com/fanone/gen-rpc/kitex_gen/interaction/v1"
	"example.com/fanone/services/interaction/internal/repository"
	"example.com/fanone/services/interaction/internal/repository/model"
	"example.com/fanone/services/interaction/internal/service"
)

type RPCHandler struct {
	store *repository.Store
}

func NewRPCHandler(store *repository.Store) *RPCHandler {
	return &RPCHandler{store: store}
}

func (h *RPCHandler) VideoLikeAction(ctx context.Context, req *interactionv1.VideoLikeActionRequest) (*interactionv1.VideoLikeActionResponse, error) {
	svc := service.NewInteractionService(h.store)
	action := service.LikeActionAdd
	if req.GetActionType() == 2 {
		action = service.LikeActionCancel
	}
	delta, err := svc.LikeVideo(ctx, uint(req.GetUserId()), uint(req.GetVideoId()), action)
	return &interactionv1.VideoLikeActionResponse{AppliedDelta: delta}, err
}

func (h *RPCHandler) ListLikedVideos(ctx context.Context, req *interactionv1.ListLikedVideosRequest) (*interactionv1.ListLikedVideosResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	svc := service.NewInteractionService(h.store)
	items, total, err := svc.ListLikedVideos(ctx, uint(req.GetUserId()), offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &interactionv1.ListLikedVideosResponse{Data: &interactionv1.VideoList{Items: videosToProto(items), Total: total}}, nil
}

func (h *RPCHandler) PublishComment(ctx context.Context, req *interactionv1.PublishCommentRequest) (*interactionv1.PublishCommentResponse, error) {
	svc := service.NewInteractionService(h.store)
	err := svc.PublishComment(ctx, uint(req.GetUserId()), uint(req.GetVideoId()), req.GetContent())
	return &interactionv1.PublishCommentResponse{}, err
}

func (h *RPCHandler) ListUserComments(ctx context.Context, req *interactionv1.ListUserCommentsRequest) (*interactionv1.ListUserCommentsResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	svc := service.NewInteractionService(h.store)
	items, total, err := svc.ListUserComments(ctx, uint(req.GetUserId()), offset, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]*interactionv1.Comment, 0, len(items))
	for i := range items {
		out = append(out, commentToProto(&items[i]))
	}
	return &interactionv1.ListUserCommentsResponse{Data: &interactionv1.CommentList{Items: out, Total: total}}, nil
}

func (h *RPCHandler) ListVideoComments(ctx context.Context, req *interactionv1.ListVideoCommentsRequest) (*interactionv1.ListVideoCommentsResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	svc := service.NewInteractionService(h.store)
	items, total, err := svc.ListVideoComments(ctx, uint(req.GetVideoId()), offset, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]*interactionv1.VideoComment, 0, len(items))
	for _, item := range items {
		out = append(out, &interactionv1.VideoComment{
			Id:        uint64(item.ID),
			UserId:    uint64(item.UserID),
			Username:  item.Username,
			AvatarUrl: item.AvatarURL,
			Content:   item.Content,
			LikeCount: item.LikeCount,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &interactionv1.ListVideoCommentsResponse{Data: &interactionv1.VideoCommentList{Items: out, Total: total}}, nil
}

func (h *RPCHandler) DeleteComment(ctx context.Context, req *interactionv1.DeleteCommentRequest) (*interactionv1.DeleteCommentResponse, error) {
	svc := service.NewInteractionService(h.store)
	videoID, err := svc.DeleteComment(ctx, uint(req.GetUserId()), uint(req.GetCommentId()))
	return &interactionv1.DeleteCommentResponse{VideoId: uint64(videoID)}, err
}

func (h *RPCHandler) RelationAction(ctx context.Context, req *interactionv1.RelationActionRequest) (*interactionv1.RelationActionResponse, error) {
	svc := service.NewRelationService(h.store)
	action := service.FollowActionFollow
	if req.GetActionType() == 2 {
		action = service.FollowActionUnfollow
	}
	_, err := svc.FollowUser(ctx, uint(req.GetUserId()), uint(req.GetToUserId()), action)
	return &interactionv1.RelationActionResponse{}, err
}

func (h *RPCHandler) ListFollowings(ctx context.Context, req *interactionv1.ListFollowingsRequest) (*interactionv1.ListFollowingsResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	svc := service.NewRelationService(h.store)
	items, total, err := svc.ListFollowings(ctx, uint(req.GetUserId()), offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &interactionv1.ListFollowingsResponse{Data: &interactionv1.SocialList{Items: usersToSocial(items), Total: total}}, nil
}

func (h *RPCHandler) ListFollowers(ctx context.Context, req *interactionv1.ListFollowersRequest) (*interactionv1.ListFollowersResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	svc := service.NewRelationService(h.store)
	items, total, err := svc.ListFollowers(ctx, uint(req.GetUserId()), offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &interactionv1.ListFollowersResponse{Data: &interactionv1.SocialList{Items: usersToSocial(items), Total: total}}, nil
}

func (h *RPCHandler) ListFriends(ctx context.Context, req *interactionv1.ListFriendsRequest) (*interactionv1.ListFriendsResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	svc := service.NewRelationService(h.store)
	items, total, err := svc.ListFriends(ctx, uint(req.GetUserId()), offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &interactionv1.ListFriendsResponse{Data: &interactionv1.SocialList{Items: usersToSocial(items), Total: total}}, nil
}

func (h *RPCHandler) SyncUser(ctx context.Context, req *interactionv1.SyncUserRequest) (*interactionv1.SyncUserResponse, error) {
	svc := service.NewInteractionService(h.store)
	if err := svc.SyncUser(ctx, &model.User{
		ID:        uint(req.GetId()),
		Username:  req.GetUsername(),
		AvatarURL: req.GetAvatarUrl(),
	}); err != nil {
		return nil, err
	}
	return &interactionv1.SyncUserResponse{}, nil
}

func (h *RPCHandler) SyncVideo(ctx context.Context, req *interactionv1.SyncVideoRequest) (*interactionv1.SyncVideoResponse, error) {
	video := req.GetVideo()
	svc := service.NewInteractionService(h.store)
	if err := svc.SyncVideo(ctx, &model.Video{
		ID:           uint(video.GetId()),
		UserID:       uint(video.GetUserId()),
		VideoURL:     video.GetVideoUrl(),
		CoverURL:     video.GetCoverUrl(),
		Title:        video.GetTitle(),
		Description:  video.GetDescription(),
		VisitCount:   video.GetVisitCount(),
		LikeCount:    video.GetLikeCount(),
		CommentCount: video.GetCommentCount(),
	}); err != nil {
		return nil, err
	}
	return &interactionv1.SyncVideoResponse{}, nil
}

func videosToProto(items []model.Video) []*interactionv1.Video {
	out := make([]*interactionv1.Video, 0, len(items))
	for i := range items {
		out = append(out, &interactionv1.Video{
			Id:           uint64(items[i].ID),
			UserId:       uint64(items[i].UserID),
			VideoUrl:     items[i].VideoURL,
			CoverUrl:     items[i].CoverURL,
			Title:        items[i].Title,
			Description:  items[i].Description,
			VisitCount:   items[i].VisitCount,
			LikeCount:    items[i].LikeCount,
			CommentCount: items[i].CommentCount,
			CreatedAt:    items[i].CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    items[i].UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return out
}

func commentToProto(c *model.Comment) *interactionv1.Comment {
	if c == nil {
		return nil
	}
	out := &interactionv1.Comment{
		Id:        uint64(c.ID),
		UserId:    uint64(c.UserID),
		VideoId:   uint64(c.VideoID),
		Content:   c.Content,
		LikeCount: c.LikeCount,
		CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if c.ParentID != nil {
		out.ParentId = uint64(*c.ParentID)
	}
	if c.DeletedAt.Valid {
		out.DeletedAt = c.DeletedAt.Time.Format("2006-01-02 15:04:05")
	}
	return out
}

func usersToSocial(items []model.User) []*interactionv1.SocialProfile {
	out := make([]*interactionv1.SocialProfile, 0, len(items))
	for i := range items {
		out = append(out, &interactionv1.SocialProfile{
			Id:        uint64(items[i].ID),
			Username:  items[i].Username,
			AvatarUrl: items[i].AvatarURL,
		})
	}
	return out
}

func normalizePage(pageNum, pageSize int32) (int, int, int) {
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	page := int(pageNum)
	size := int(pageSize)
	return page, size, (page - 1) * size
}
