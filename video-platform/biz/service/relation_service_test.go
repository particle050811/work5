package service

import (
	"context"
	"errors"
	"testing"

	"video-platform/biz/dal/model"

	"gorm.io/gorm"
)

func TestRelationServiceFollowUserLifecycle(t *testing.T) {
	store, gdb := newTestStore(t)
	svc := NewRelationService(store)
	alice := createTestUser(t, gdb, "alice")
	bob := createTestUser(t, gdb, "bob")

	if _, err := svc.FollowUser(context.Background(), alice.ID, alice.ID, FollowActionFollow); !errors.Is(err, ErrCannotFollowSelf) {
		t.Fatalf("关注自己错误 = %v, want %v", err, ErrCannotFollowSelf)
	}

	delta, err := svc.FollowUser(context.Background(), alice.ID, bob.ID, FollowActionFollow)
	if err != nil {
		t.Fatalf("首次关注失败: %v", err)
	}
	if delta != 1 {
		t.Fatalf("首次关注 delta = %d, want 1", delta)
	}

	assertFollowCounts(t, gdb, alice.ID, 1, 0)
	assertFollowCounts(t, gdb, bob.ID, 0, 1)

	delta, err = svc.FollowUser(context.Background(), alice.ID, bob.ID, FollowActionFollow)
	if err != nil {
		t.Fatalf("重复关注失败: %v", err)
	}
	if delta != 0 {
		t.Fatalf("重复关注 delta = %d, want 0", delta)
	}

	delta, err = svc.FollowUser(context.Background(), alice.ID, bob.ID, FollowActionUnfollow)
	if err != nil {
		t.Fatalf("取消关注失败: %v", err)
	}
	if delta != -1 {
		t.Fatalf("取消关注 delta = %d, want -1", delta)
	}

	assertFollowCounts(t, gdb, alice.ID, 0, 0)
	assertFollowCounts(t, gdb, bob.ID, 0, 0)

	delta, err = svc.FollowUser(context.Background(), alice.ID, bob.ID, FollowActionFollow)
	if err != nil {
		t.Fatalf("恢复关注失败: %v", err)
	}
	if delta != 1 {
		t.Fatalf("恢复关注 delta = %d, want 1", delta)
	}

	var totalFollows int64
	if err := gdb.Unscoped().Model(&model.Follow{}).Where("follower_id = ? AND following_id = ?", alice.ID, bob.ID).Count(&totalFollows).Error; err != nil {
		t.Fatalf("查询关注记录失败: %v", err)
	}
	if totalFollows != 1 {
		t.Fatalf("恢复关注后总记录数 = %d, want 1", totalFollows)
	}
}

func TestRelationServiceListFriendsReturnsMutualFollows(t *testing.T) {
	store, gdb := newTestStore(t)
	svc := NewRelationService(store)
	alice := createTestUser(t, gdb, "friend_alice")
	bob := createTestUser(t, gdb, "friend_bob")
	carol := createTestUser(t, gdb, "friend_carol")

	if _, err := svc.FollowUser(context.Background(), alice.ID, bob.ID, FollowActionFollow); err != nil {
		t.Fatalf("alice 关注 bob 失败: %v", err)
	}
	if _, err := svc.FollowUser(context.Background(), bob.ID, alice.ID, FollowActionFollow); err != nil {
		t.Fatalf("bob 关注 alice 失败: %v", err)
	}
	if _, err := svc.FollowUser(context.Background(), alice.ID, carol.ID, FollowActionFollow); err != nil {
		t.Fatalf("alice 关注 carol 失败: %v", err)
	}

	friends, total, err := svc.ListFriends(context.Background(), alice.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListFriends() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("ListFriends() total = %d, want 1", total)
	}
	if len(friends) != 1 || friends[0].ID != bob.ID {
		t.Fatalf("ListFriends() = %+v, want only bob", friends)
	}
}

func assertFollowCounts(t *testing.T, gdb *gorm.DB, userID uint, wantFollowing, wantFollower int64) {
	t.Helper()

	var user model.User
	if err := gdb.First(&user, userID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if user.FollowingCount != wantFollowing || user.FollowerCount != wantFollower {
		t.Fatalf("user_id=%d following_count=%d follower_count=%d, want %d/%d", userID, user.FollowingCount, user.FollowerCount, wantFollowing, wantFollower)
	}
}
