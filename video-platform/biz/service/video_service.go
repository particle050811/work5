package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"video-platform/biz/dal"
	"video-platform/biz/dal/db"
	"video-platform/biz/dal/model"
	"video-platform/pkg/util"

	"github.com/redis/go-redis/v9"
)

const (
	hotVideosKey            = "fanone:video:hot:zset"
	hotVideosEmptyKey       = "fanone:video:hot:empty"
	hotVideosRebuildLockKey = "fanone:video:hot:rebuild:lock"
	cacheTTL                = 5 * time.Minute
	emptyCacheTTL           = 30 * time.Second
	cacheTTLJitterMax       = 90 * time.Second
	rebuildLockTTL          = 15 * time.Second
	rebuildRetryDelay       = 120 * time.Millisecond
	topN                    = 200
)

// VideoService 视频服务
type VideoService struct {
	store *dal.Store
}

// NewVideoService 创建视频服务实例
func NewVideoService(store *dal.Store) *VideoService {
	return &VideoService{store: store}
}

// CreateVideo 创建视频
func (s *VideoService) CreateVideo(ctx context.Context, video *model.Video) error {
	if err := db.CreateVideo(s.store, video); err != nil {
		return fmt.Errorf("创建视频记录失败: %w", err)
	}
	return nil
}

// GetVideoByID 根据 ID 获取视频
func (s *VideoService) GetVideoByID(ctx context.Context, videoID uint) (*model.Video, error) {
	video, err := db.GetVideoByID(s.store, videoID)
	if err != nil {
		return nil, fmt.Errorf("查询视频失败: %w", err)
	}
	if video == nil {
		return nil, ErrVideoNotFound
	}
	return video, nil
}

// ListVideosByUser 获取用户发布的视频列表
func (s *VideoService) ListVideosByUser(ctx context.Context, userID uint, offset, limit int) ([]model.Video, int64, error) {
	return db.ListVideosByUser(s.store, userID, offset, limit)
}

// SearchVideosParams 搜索视频参数
type SearchVideosParams struct {
	Keywords  string
	Username  string
	FromDate  *time.Time
	ToDate    *time.Time
	SortByHot bool
}

// SearchVideos 搜索视频
func (s *VideoService) SearchVideos(ctx context.Context, params SearchVideosParams, offset, limit int) ([]model.Video, int64, error) {
	return db.SearchVideos(s.store, db.SearchVideosParams{
		Keywords:  params.Keywords,
		Username:  params.Username,
		FromDate:  params.FromDate,
		ToDate:    params.ToDate,
		SortByHot: params.SortByHot,
	}, offset, limit)
}

// ListVideoComments 获取视频评论列表
func (s *VideoService) ListVideoComments(ctx context.Context, videoID uint, offset, limit int) ([]db.VideoCommentWithUser, int64, error) {
	return db.ListTopLevelCommentsByVideo(s.store, videoID, offset, limit)
}

// GetHotVideos 获取热门视频列表
func (s *VideoService) GetHotVideos(ctx context.Context, offset, limit int) ([]model.Video, int64, error) {
	if !s.store.HasRedis() {
		return s.listHotVideosFromDB(offset, limit)
	}

	var total int64
	if err := s.store.DB().Model(&model.Video{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	redisCli := s.store.Redis()
	ready, emptyResult, err := s.ensureHotVideosCache(ctx, redisCli)
	if err != nil {
		log.Printf("[视频模块][热门排行榜] 热榜缓存检查失败，回退数据库排序 key=%s: %v", hotVideosKey, err)
		return s.listHotVideosFromDB(offset, limit)
	}
	if emptyResult {
		return []model.Video{}, total, nil
	}
	if !ready {
		return s.listHotVideosFromDB(offset, limit)
	}

	zs, err := redisCli.ZRevRangeWithScores(ctx, hotVideosKey, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		log.Printf("[视频模块][热门排行榜] Redis 读取热榜失败，回退数据库排序 key=%s: %v", hotVideosKey, err)
		return s.listHotVideosFromDB(offset, limit)
	}

	ids := make([]uint, 0, len(zs))
	for _, z := range zs {
		idStr, ok := z.Member.(string)
		if !ok {
			continue
		}
		id, err := util.ParseUint(idStr)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	videos, err := db.GetVideosByIDs(s.store, ids)
	if err != nil {
		return nil, 0, err
	}

	// 按热度排序
	byID := make(map[uint]model.Video, len(videos))
	for i := range videos {
		byID[videos[i].ID] = videos[i]
	}
	ordered := make([]model.Video, 0, len(ids))
	for _, id := range ids {
		if v, ok := byID[id]; ok {
			ordered = append(ordered, v)
		}
	}
	return ordered, total, nil
}

func (s *VideoService) listHotVideosFromDB(offset, limit int) ([]model.Video, int64, error) {
	var total int64
	base := s.store.DB().Model(&model.Video{})
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var videos []model.Video
	if err := base.
		Order(hotScoreOrderSQL()).
		Order("created_at desc").
		Offset(offset).
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

func (s *VideoService) ensureHotVideosCache(ctx context.Context, redisCli dal.RedisClient) (bool, bool, error) {
	exists, err := redisCli.Exists(ctx, hotVideosKey, hotVideosEmptyKey).Result()
	if err != nil {
		return false, false, err
	}
	if exists > 0 {
		emptyExists, err := redisCli.Exists(ctx, hotVideosEmptyKey).Result()
		if err != nil {
			return false, false, err
		}
		return true, emptyExists > 0, nil
	}

	locked, err := redisCli.SetNX(ctx, hotVideosRebuildLockKey, "1", rebuildLockTTL).Result()
	if err != nil {
		return false, false, err
	}
	if locked {
		defer func() {
			if delErr := redisCli.Del(ctx, hotVideosRebuildLockKey).Err(); delErr != nil {
				log.Printf("[视频模块][热门排行榜] 释放重建锁失败 key=%s: %v", hotVideosRebuildLockKey, delErr)
			}
		}()
		emptyResult, rebuildErr := s.rebuildHotVideosCache(ctx, redisCli)
		return rebuildErr == nil, emptyResult, rebuildErr
	}

	time.Sleep(rebuildRetryDelay)
	exists, err = redisCli.Exists(ctx, hotVideosKey, hotVideosEmptyKey).Result()
	if err != nil {
		return false, false, err
	}
	if exists == 0 {
		return false, false, nil
	}
	emptyExists, err := redisCli.Exists(ctx, hotVideosEmptyKey).Result()
	if err != nil {
		return false, false, err
	}
	return true, emptyExists > 0, nil
}

// rebuildHotVideosZSet 重建热榜缓存
func (s *VideoService) rebuildHotVideosCache(ctx context.Context, redisCli dal.RedisClient) (bool, error) {
	var top []model.Video
	if err := s.store.DB().
		Model(&model.Video{}).
		Order(hotScoreOrderSQL()).
		Order("created_at desc").
		Limit(topN).
		Find(&top).Error; err != nil {
		return false, err
	}

	if err := redisCli.Del(ctx, hotVideosKey, hotVideosEmptyKey).Err(); err != nil {
		return false, err
	}
	if len(top) == 0 {
		if err := redisCli.Set(ctx, hotVideosEmptyKey, "1", emptyCacheTTLWithJitter()).Err(); err != nil {
			return false, err
		}
		return true, nil
	}

	zs := make([]redis.Z, 0, len(top))
	for i := range top {
		score := calculateVideoHotScore(top[i])
		zs = append(zs, redis.Z{
			Score:  score,
			Member: fmt.Sprintf("%d", top[i].ID),
		})
	}

	if err := redisCli.ZAdd(ctx, hotVideosKey, zs...).Err(); err != nil {
		return false, err
	}
	if err := redisCli.Expire(ctx, hotVideosKey, cacheTTLWithJitter()).Err(); err != nil {
		return false, err
	}
	return false, nil
}

func calculateVideoHotScore(video model.Video) float64 {
	return float64(video.LikeCount*3 + video.CommentCount*2 + video.VisitCount)
}

func hotScoreOrderSQL() string {
	return "(like_count * 3 + comment_count * 2 + visit_count) desc"
}

func cacheTTLWithJitter() time.Duration {
	return cacheTTL + time.Duration(rand.Int63n(int64(cacheTTLJitterMax)+1))
}

func emptyCacheTTLWithJitter() time.Duration {
	return emptyCacheTTL + time.Duration(rand.Int63n(int64(emptyCacheTTL)/2+1))
}
