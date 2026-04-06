package main

import (
	"fmt"

	interactionv1 "example.com/fanone/gen-rpc/kitex_gen/interaction/v1"
	userrpcv1 "example.com/fanone/gen-rpc/kitex_gen/user/v1"
	videorpcv1 "example.com/fanone/gen-rpc/kitex_gen/video/v1"
	api "example.com/fanone/work5/idl/http/gen/v1"
)

func rpcUserToHTTP(user *userrpcv1.UserProfile) *api.User {
	if user == nil {
		return nil
	}
	return &api.User{
		Id:        fmt.Sprintf("%d", user.GetId()),
		Username:  user.GetUsername(),
		AvatarUrl: user.GetAvatarUrl(),
		CreatedAt: user.GetCreatedAt(),
		UpdatedAt: user.GetUpdatedAt(),
		DeletedAt: user.GetDeletedAt(),
	}
}

func rpcVideoListToHTTP(data *videorpcv1.VideoList) *api.VideoListWithTotal {
	if data == nil {
		return &api.VideoListWithTotal{}
	}
	items := make([]*api.Video, 0, len(data.GetItems()))
	for _, item := range data.GetItems() {
		items = append(items, &api.Video{
			Id:           fmt.Sprintf("%d", item.GetId()),
			UserId:       fmt.Sprintf("%d", item.GetUserId()),
			VideoUrl:     item.GetVideoUrl(),
			CoverUrl:     item.GetCoverUrl(),
			Title:        item.GetTitle(),
			Description:  item.GetDescription(),
			VisitCount:   item.GetVisitCount(),
			LikeCount:    item.GetLikeCount(),
			CommentCount: item.GetCommentCount(),
			CreatedAt:    item.GetCreatedAt(),
			UpdatedAt:    item.GetUpdatedAt(),
			DeletedAt:    item.GetDeletedAt(),
		})
	}
	return &api.VideoListWithTotal{Items: items, Total: data.GetTotal()}
}

func interactionVideoListToHTTP(data *interactionv1.VideoList) *api.VideoListWithTotal {
	if data == nil {
		return &api.VideoListWithTotal{}
	}
	items := make([]*api.Video, 0, len(data.GetItems()))
	for _, item := range data.GetItems() {
		items = append(items, &api.Video{
			Id:           fmt.Sprintf("%d", item.GetId()),
			UserId:       fmt.Sprintf("%d", item.GetUserId()),
			VideoUrl:     item.GetVideoUrl(),
			CoverUrl:     item.GetCoverUrl(),
			Title:        item.GetTitle(),
			Description:  item.GetDescription(),
			VisitCount:   item.GetVisitCount(),
			LikeCount:    item.GetLikeCount(),
			CommentCount: item.GetCommentCount(),
			CreatedAt:    item.GetCreatedAt(),
			UpdatedAt:    item.GetUpdatedAt(),
			DeletedAt:    item.GetDeletedAt(),
		})
	}
	return &api.VideoListWithTotal{Items: items, Total: data.GetTotal()}
}

func interactionVideoFromVideoRPC(video *videorpcv1.Video) *interactionv1.Video {
	if video == nil {
		return nil
	}
	return &interactionv1.Video{
		Id:           video.GetId(),
		UserId:       video.GetUserId(),
		VideoUrl:     video.GetVideoUrl(),
		CoverUrl:     video.GetCoverUrl(),
		Title:        video.GetTitle(),
		Description:  video.GetDescription(),
		VisitCount:   video.GetVisitCount(),
		LikeCount:    video.GetLikeCount(),
		CommentCount: video.GetCommentCount(),
		CreatedAt:    video.GetCreatedAt(),
		UpdatedAt:    video.GetUpdatedAt(),
		DeletedAt:    video.GetDeletedAt(),
	}
}

func socialListToHTTP(data *interactionv1.SocialList) *api.SocialListWithTotal {
	if data == nil {
		return &api.SocialListWithTotal{}
	}
	items := make([]*api.SocialProfile, 0, len(data.GetItems()))
	for _, item := range data.GetItems() {
		items = append(items, &api.SocialProfile{
			Id:        fmt.Sprintf("%d", item.GetId()),
			Username:  item.GetUsername(),
			AvatarUrl: item.GetAvatarUrl(),
		})
	}
	return &api.SocialListWithTotal{Items: items, Total: data.GetTotal()}
}

func idToString(id uint64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("%d", id)
}
