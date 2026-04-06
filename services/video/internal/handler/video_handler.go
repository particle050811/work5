package handler

import (
	"context"
	"strings"
	"time"

	videov1 "example.com/fanone/gen-rpc/kitex_gen/video/v1"
	"example.com/fanone/services/video/internal/repository"
	"example.com/fanone/services/video/internal/repository/model"
	"example.com/fanone/services/video/internal/service"
)

type RPCHandler struct {
	store  *repository.Store
	syncer service.InteractionVideoSyncer
}

func NewRPCHandler(store *repository.Store, syncer service.InteractionVideoSyncer) *RPCHandler {
	return &RPCHandler{store: store, syncer: syncer}
}

func (h *RPCHandler) CreateVideo(ctx context.Context, req *videov1.CreateVideoRequest) (*videov1.CreateVideoResponse, error) {
	svc := service.NewVideoService(h.store, h.syncer)
	video := &model.Video{
		UserID:      uint(req.GetUserId()),
		VideoURL:    req.GetVideoUrl(),
		CoverURL:    req.GetCoverUrl(),
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
	}
	if err := svc.CreateVideo(ctx, video); err != nil {
		return nil, err
	}
	return &videov1.CreateVideoResponse{Video: modelToProtoVideo(video)}, nil
}

func (h *RPCHandler) ListPublishedVideos(ctx context.Context, req *videov1.ListPublishedVideosRequest) (*videov1.ListPublishedVideosResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	svc := service.NewVideoService(h.store, h.syncer)
	items, total, err := svc.ListVideosByUser(ctx, uint(req.GetUserId()), offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &videov1.ListPublishedVideosResponse{Data: &videov1.VideoList{Items: videosToProto(items), Total: total}}, nil
}

func (h *RPCHandler) SearchVideos(ctx context.Context, req *videov1.SearchVideosRequest) (*videov1.SearchVideosResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	from, to := parseUnixRange(req.GetFromDate(), req.GetToDate())
	svc := service.NewVideoService(h.store, h.syncer)
	items, total, err := svc.SearchVideos(ctx, service.SearchVideosParams{
		Keywords:  strings.TrimSpace(req.GetKeywords()),
		Username:  strings.TrimSpace(req.GetUsername()),
		FromDate:  from,
		ToDate:    to,
		SortByHot: strings.EqualFold(strings.TrimSpace(req.GetSortBy()), "hot"),
	}, offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &videov1.SearchVideosResponse{Data: &videov1.VideoList{Items: videosToProto(items), Total: total}}, nil
}

func (h *RPCHandler) GetHotVideos(ctx context.Context, req *videov1.GetHotVideosRequest) (*videov1.GetHotVideosResponse, error) {
	_, pageSize, offset := normalizePage(req.GetPageNum(), req.GetPageSize())
	svc := service.NewVideoService(h.store, h.syncer)
	items, total, err := svc.GetHotVideos(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &videov1.GetHotVideosResponse{Data: &videov1.VideoList{Items: videosToProto(items), Total: total}}, nil
}

func (h *RPCHandler) SyncUser(ctx context.Context, req *videov1.SyncUserRequest) (*videov1.SyncUserResponse, error) {
	svc := service.NewVideoService(h.store, h.syncer)
	if err := svc.SyncUser(ctx, &model.User{
		ID:        uint(req.GetId()),
		Username:  req.GetUsername(),
		AvatarURL: req.GetAvatarUrl(),
	}); err != nil {
		return nil, err
	}
	return &videov1.SyncUserResponse{}, nil
}

func (h *RPCHandler) SyncVideoCounters(ctx context.Context, req *videov1.SyncVideoCountersRequest) (*videov1.SyncVideoCountersResponse, error) {
	svc := service.NewVideoService(h.store, h.syncer)
	if err := svc.SyncVideoCounters(ctx, uint(req.GetVideoId()), req.GetVisitDelta(), req.GetLikeDelta(), req.GetCommentDelta()); err != nil {
		return nil, err
	}
	return &videov1.SyncVideoCountersResponse{}, nil
}

func videosToProto(items []model.Video) []*videov1.Video {
	out := make([]*videov1.Video, 0, len(items))
	for i := range items {
		out = append(out, modelToProtoVideo(&items[i]))
	}
	return out
}

func modelToProtoVideo(v *model.Video) *videov1.Video {
	if v == nil {
		return nil
	}
	out := &videov1.Video{
		Id:           uint64(v.ID),
		UserId:       uint64(v.UserID),
		VideoUrl:     v.VideoURL,
		CoverUrl:     v.CoverURL,
		Title:        v.Title,
		Description:  v.Description,
		VisitCount:   v.VisitCount,
		LikeCount:    v.LikeCount,
		CommentCount: v.CommentCount,
		CreatedAt:    v.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    v.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if v.DeletedAt.Valid {
		out.DeletedAt = v.DeletedAt.Time.Format("2006-01-02 15:04:05")
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

func parseUnixRange(fromUnix, toUnix int64) (*time.Time, *time.Time) {
	var from *time.Time
	var to *time.Time
	if fromUnix > 0 {
		t := time.Unix(fromUnix, 0)
		from = &t
	}
	if toUnix > 0 {
		t := time.Unix(toUnix, 0)
		to = &t
	}
	return from, to
}
