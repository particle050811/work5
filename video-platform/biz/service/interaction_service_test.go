package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"video-platform/biz/dal/model"
)

func TestInteractionServiceLikeVideoLifecycle(t *testing.T) {
	store, gdb := newTestStore(t)
	svc := NewInteractionService(store)
	user := createTestUser(t, gdb, "like_user")
	video := createTestVideo(t, gdb, user.ID, "like video")

	delta, err := svc.LikeVideo(context.Background(), user.ID, video.ID, LikeActionAdd)
	if err != nil {
		t.Fatalf("首次点赞失败: %v", err)
	}
	if delta != 1 {
		t.Fatalf("首次点赞 delta = %d, want 1", delta)
	}

	var likeCount int64
	if err := gdb.Model(&model.VideoLike{}).Where("user_id = ? AND video_id = ?", user.ID, video.ID).Count(&likeCount).Error; err != nil {
		t.Fatalf("查询点赞记录失败: %v", err)
	}
	if likeCount != 1 {
		t.Fatalf("点赞记录数 = %d, want 1", likeCount)
	}

	var videoAfterLike model.Video
	if err := gdb.First(&videoAfterLike, video.ID).Error; err != nil {
		t.Fatalf("查询视频失败: %v", err)
	}
	if videoAfterLike.LikeCount != 1 {
		t.Fatalf("点赞后 like_count = %d, want 1", videoAfterLike.LikeCount)
	}

	delta, err = svc.LikeVideo(context.Background(), user.ID, video.ID, LikeActionAdd)
	if err != nil {
		t.Fatalf("重复点赞失败: %v", err)
	}
	if delta != 0 {
		t.Fatalf("重复点赞 delta = %d, want 0", delta)
	}

	delta, err = svc.LikeVideo(context.Background(), user.ID, video.ID, LikeActionCancel)
	if err != nil {
		t.Fatalf("取消点赞失败: %v", err)
	}
	if delta != -1 {
		t.Fatalf("取消点赞 delta = %d, want -1", delta)
	}

	var videoAfterCancel model.Video
	if err := gdb.First(&videoAfterCancel, video.ID).Error; err != nil {
		t.Fatalf("查询视频失败: %v", err)
	}
	if videoAfterCancel.LikeCount != 0 {
		t.Fatalf("取消点赞后 like_count = %d, want 0", videoAfterCancel.LikeCount)
	}

	delta, err = svc.LikeVideo(context.Background(), user.ID, video.ID, LikeActionAdd)
	if err != nil {
		t.Fatalf("恢复点赞失败: %v", err)
	}
	if delta != 1 {
		t.Fatalf("恢复点赞 delta = %d, want 1", delta)
	}

	var totalLikes int64
	if err := gdb.Unscoped().Model(&model.VideoLike{}).Where("user_id = ? AND video_id = ?", user.ID, video.ID).Count(&totalLikes).Error; err != nil {
		t.Fatalf("查询全部点赞记录失败: %v", err)
	}
	if totalLikes != 1 {
		t.Fatalf("恢复点赞后总记录数 = %d, want 1", totalLikes)
	}
}

func TestInteractionServicePublishAndDeleteComment(t *testing.T) {
	store, gdb := newTestStore(t)
	svc := NewInteractionService(store)
	author := createTestUser(t, gdb, "comment_author")
	other := createTestUser(t, gdb, "comment_other")
	video := createTestVideo(t, gdb, author.ID, "comment video")

	if err := svc.PublishComment(context.Background(), author.ID, video.ID, "   "); !errors.Is(err, ErrCommentEmpty) {
		t.Fatalf("空评论错误 = %v, want %v", err, ErrCommentEmpty)
	}

	longContent := strings.Repeat("a", 1001)
	if err := svc.PublishComment(context.Background(), author.ID, video.ID, longContent); !errors.Is(err, ErrCommentTooLong) {
		t.Fatalf("长评论错误 = %v, want %v", err, ErrCommentTooLong)
	}

	if err := svc.PublishComment(context.Background(), author.ID, video.ID, "  第一条评论  "); err != nil {
		t.Fatalf("发布评论失败: %v", err)
	}

	var comment model.Comment
	if err := gdb.First(&comment).Error; err != nil {
		t.Fatalf("查询评论失败: %v", err)
	}
	if comment.Content != "第一条评论" {
		t.Fatalf("评论内容 = %q, want %q", comment.Content, "第一条评论")
	}

	var videoAfterComment model.Video
	if err := gdb.First(&videoAfterComment, video.ID).Error; err != nil {
		t.Fatalf("查询视频失败: %v", err)
	}
	if videoAfterComment.CommentCount != 1 {
		t.Fatalf("发布评论后 comment_count = %d, want 1", videoAfterComment.CommentCount)
	}

	if err := svc.DeleteComment(context.Background(), other.ID, comment.ID); !errors.Is(err, ErrNoPermission) {
		t.Fatalf("非作者删除错误 = %v, want %v", err, ErrNoPermission)
	}

	if err := svc.DeleteComment(context.Background(), author.ID, comment.ID); err != nil {
		t.Fatalf("作者删除评论失败: %v", err)
	}

	var deletedComment model.Comment
	if err := gdb.Unscoped().First(&deletedComment, comment.ID).Error; err != nil {
		t.Fatalf("查询已删除评论失败: %v", err)
	}
	if !deletedComment.DeletedAt.Valid {
		t.Fatal("评论未被软删除")
	}

	var videoAfterDelete model.Video
	if err := gdb.First(&videoAfterDelete, video.ID).Error; err != nil {
		t.Fatalf("查询视频失败: %v", err)
	}
	if videoAfterDelete.CommentCount != 0 {
		t.Fatalf("删除评论后 comment_count = %d, want 0", videoAfterDelete.CommentCount)
	}
}

func TestInteractionServiceDeleteCommentNotFound(t *testing.T) {
	store, _ := newTestStore(t)
	svc := NewInteractionService(store)

	err := svc.DeleteComment(context.Background(), 1, 999)
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("DeleteComment() error = %v, want %v", err, ErrCommentNotFound)
	}
}
