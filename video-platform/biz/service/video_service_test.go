package service

import (
	"context"
	"testing"
	"time"

	"video-platform/biz/dal"
	"video-platform/biz/dal/model"

	"github.com/redis/go-redis/v9"
)

func TestCalculateVideoHotScore(t *testing.T) {
	video := model.Video{
		VisitCount:   11,
		LikeCount:    4,
		CommentCount: 3,
	}

	got := calculateVideoHotScore(video)
	want := float64(4*3 + 3*2 + 11)
	if got != want {
		t.Fatalf("calculateVideoHotScore() = %v, want %v", got, want)
	}
}

func TestListHotVideosFromDBUsesHotScore(t *testing.T) {
	store, gdb := newTestStore(t)
	user := createTestUser(t, gdb, "hot-score-user")

	highVisit := createTestVideo(t, gdb, user.ID, "high-visit")
	strongLike := createTestVideo(t, gdb, user.ID, "strong-like")

	if err := gdb.Model(&model.Video{}).Where("id = ?", highVisit.ID).Updates(map[string]interface{}{
		"visit_count":   10,
		"like_count":    0,
		"comment_count": 0,
	}).Error; err != nil {
		t.Fatalf("更新 highVisit 失败: %v", err)
	}
	if err := gdb.Model(&model.Video{}).Where("id = ?", strongLike.ID).Updates(map[string]interface{}{
		"visit_count":   1,
		"like_count":    4,
		"comment_count": 0,
	}).Error; err != nil {
		t.Fatalf("更新 strongLike 失败: %v", err)
	}

	service := NewVideoService(store)
	videos, total, err := service.listHotVideosFromDB(0, 10)
	if err != nil {
		t.Fatalf("listHotVideosFromDB 返回错误: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(videos) != 2 {
		t.Fatalf("len(videos) = %d, want 2", len(videos))
	}
	if videos[0].ID != strongLike.ID {
		t.Fatalf("第一个视频 = %d, want %d", videos[0].ID, strongLike.ID)
	}
}

func TestEnsureHotVideosCacheWithEmptyMarker(t *testing.T) {
	store, _ := newTestStore(t)
	service := NewVideoService(store)
	redisCli := newFakeRedisClient()
	redisCli.exists[hotVideosEmptyKey] = true

	ready, emptyResult, err := service.ensureHotVideosCache(context.Background(), redisCli)
	if err != nil {
		t.Fatalf("ensureHotVideosCache 返回错误: %v", err)
	}
	if !ready {
		t.Fatalf("ready = false, want true")
	}
	if !emptyResult {
		t.Fatalf("emptyResult = false, want true")
	}
}

func TestEnsureHotVideosCacheRebuildsEmptyResult(t *testing.T) {
	store, _ := newTestStore(t)
	service := NewVideoService(store)
	redisCli := newFakeRedisClient()

	ready, emptyResult, err := service.ensureHotVideosCache(context.Background(), redisCli)
	if err != nil {
		t.Fatalf("ensureHotVideosCache 返回错误: %v", err)
	}
	if !ready {
		t.Fatalf("ready = false, want true")
	}
	if !emptyResult {
		t.Fatalf("emptyResult = false, want true")
	}
	if !redisCli.exists[hotVideosEmptyKey] {
		t.Fatalf("empty marker 未写入")
	}
	if redisCli.setValue[hotVideosEmptyKey] != "1" {
		t.Fatalf("empty marker 值 = %q, want 1", redisCli.setValue[hotVideosEmptyKey])
	}
	if redisCli.lastSetTTL < emptyCacheTTL {
		t.Fatalf("empty marker TTL = %v, want >= %v", redisCli.lastSetTTL, emptyCacheTTL)
	}
}

func TestEnsureHotVideosCacheReturnsFallbackWhenLockHeld(t *testing.T) {
	store, _ := newTestStore(t)
	service := NewVideoService(store)
	redisCli := newFakeRedisClient()
	redisCli.setNXResult = false

	ready, emptyResult, err := service.ensureHotVideosCache(context.Background(), redisCli)
	if err != nil {
		t.Fatalf("ensureHotVideosCache 返回错误: %v", err)
	}
	if ready {
		t.Fatalf("ready = true, want false")
	}
	if emptyResult {
		t.Fatalf("emptyResult = true, want false")
	}
}

func TestRebuildHotVideosCacheSetsJitterTTL(t *testing.T) {
	store, gdb := newTestStore(t)
	user := createTestUser(t, gdb, "hot-ttl-user")
	video := createTestVideo(t, gdb, user.ID, "ttl-video")
	if err := gdb.Model(&model.Video{}).Where("id = ?", video.ID).Updates(map[string]interface{}{
		"visit_count":   8,
		"like_count":    2,
		"comment_count": 1,
	}).Error; err != nil {
		t.Fatalf("更新视频热度字段失败: %v", err)
	}

	service := NewVideoService(store)
	redisCli := newFakeRedisClient()
	emptyResult, err := service.rebuildHotVideosCache(context.Background(), redisCli)
	if err != nil {
		t.Fatalf("rebuildHotVideosCache 返回错误: %v", err)
	}
	if emptyResult {
		t.Fatalf("emptyResult = true, want false")
	}
	if !redisCli.exists[hotVideosKey] {
		t.Fatalf("热榜 key 未写入")
	}
	if redisCli.lastExpireTTL < cacheTTL || redisCli.lastExpireTTL > cacheTTL+cacheTTLJitterMax {
		t.Fatalf("热榜 TTL = %v, want in [%v, %v]", redisCli.lastExpireTTL, cacheTTL, cacheTTL+cacheTTLJitterMax)
	}
	if len(redisCli.zMembers[hotVideosKey]) != 1 {
		t.Fatalf("热榜成员数 = %d, want 1", len(redisCli.zMembers[hotVideosKey]))
	}
	member := redisCli.zMembers[hotVideosKey][0]
	if member.Score != calculateVideoHotScore(model.Video{VisitCount: 8, LikeCount: 2, CommentCount: 1}) {
		t.Fatalf("热榜分数 = %v, want %v", member.Score, calculateVideoHotScore(model.Video{VisitCount: 8, LikeCount: 2, CommentCount: 1}))
	}
}

type fakeRedisClient struct {
	exists        map[string]bool
	setValue      map[string]string
	zMembers      map[string][]redis.Z
	setNXResult   bool
	lastSetTTL    time.Duration
	lastExpireTTL time.Duration
}

func newFakeRedisClient() *fakeRedisClient {
	return &fakeRedisClient{
		exists:      make(map[string]bool),
		setValue:    make(map[string]string),
		zMembers:    make(map[string][]redis.Z),
		setNXResult: true,
	}
}

func (f *fakeRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	if !f.exists[key] {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(f.setValue[key], nil)
}

func (f *fakeRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	f.exists[key] = true
	f.setValue[key] = toString(value)
	f.lastSetTTL = expiration
	return redis.NewStatusResult("OK", nil)
}

func (f *fakeRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	if !f.setNXResult {
		return redis.NewBoolResult(false, nil)
	}
	f.exists[key] = true
	f.setValue[key] = toString(value)
	return redis.NewBoolResult(true, nil)
}

func (f *fakeRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	var deleted int64
	for _, key := range keys {
		if f.exists[key] {
			deleted++
		}
		delete(f.exists, key)
		delete(f.setValue, key)
		delete(f.zMembers, key)
	}
	return redis.NewIntResult(deleted, nil)
}

func (f *fakeRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	var count int64
	for _, key := range keys {
		if f.exists[key] {
			count++
		}
	}
	return redis.NewIntResult(count, nil)
}

func (f *fakeRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	if f.exists[key] {
		f.lastExpireTTL = expiration
		return redis.NewBoolResult(true, nil)
	}
	return redis.NewBoolResult(false, nil)
}

func (f *fakeRedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	f.exists[key] = true
	f.zMembers[key] = append([]redis.Z(nil), members...)
	return redis.NewIntResult(int64(len(members)), nil)
}

func (f *fakeRedisClient) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
	members := f.zMembers[key]
	cmd := redis.NewZSliceCmd(ctx)
	if start >= int64(len(members)) {
		cmd.SetVal([]redis.Z{})
		return cmd
	}
	if stop >= int64(len(members)) {
		stop = int64(len(members) - 1)
	}
	cmd.SetVal(members[start : stop+1])
	return cmd
}

func (f *fakeRedisClient) ZIncrBy(ctx context.Context, key string, increment float64, member string) *redis.FloatCmd {
	return redis.NewFloatResult(increment, nil)
}

func (f *fakeRedisClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (f *fakeRedisClient) ZScore(ctx context.Context, key string, member string) *redis.FloatCmd {
	return redis.NewFloatResult(0, nil)
}

func toString(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

var _ dal.RedisClient = (*fakeRedisClient)(nil)
