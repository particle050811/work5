package db

import (
	"errors"
	"time"

	"video-platform/biz/dal/model"

	"gorm.io/gorm"
)

// CreateVideo 创建视频记录
func CreateVideo(store DBProvider, video *model.Video) error {
	return store.DB().Create(video).Error
}

// ListVideosByUser 分页获取某用户发布的视频
func ListVideosByUser(store DBProvider, userID uint, offset, limit int) ([]model.Video, int64, error) {
	var total int64
	base := store.DB().Model(&model.Video{}).Where("user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var videos []model.Video
	if err := base.Order("created_at desc").Offset(offset).Limit(limit).Find(&videos).Error; err != nil {
		return nil, 0, err
	}
	return videos, total, nil
}

type SearchVideosParams struct {
	Keywords  string
	Username  string
	FromDate  *time.Time
	ToDate    *time.Time
	SortByHot bool
}

// SearchVideos 搜索视频（AND 关系）
func SearchVideos(store DBProvider, p SearchVideosParams, offset, limit int) ([]model.Video, int64, error) {
	tx := store.DB().Model(&model.Video{})

	// 关键词（标题 + 简介）
	if p.Keywords != "" {
		kw := "%" + p.Keywords + "%"
		tx = tx.Where("(videos.title LIKE ? OR videos.description LIKE ?)", kw, kw)
	}

	// 时间区间（按 created_at）
	if p.FromDate != nil {
		tx = tx.Where("videos.created_at >= ?", *p.FromDate)
	}
	if p.ToDate != nil {
		tx = tx.Where("videos.created_at <= ?", *p.ToDate)
	}

	// 作者用户名筛选
	if p.Username != "" {
		tx = tx.Joins("JOIN users ON users.id = videos.user_id").Where("users.username = ?", p.Username)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := "videos.created_at desc"
	if p.SortByHot {
		// 简化热度：点赞*3 + 评论*2 + 访问*1
		orderBy = "(videos.like_count*3 + videos.comment_count*2 + videos.visit_count) desc"
	}

	var videos []model.Video
	if err := tx.Order(orderBy).Offset(offset).Limit(limit).Find(&videos).Error; err != nil {
		return nil, 0, err
	}
	return videos, total, nil
}

// GetVideosByIDs 按 ID 批量查询视频
func GetVideosByIDs(store DBProvider, ids []uint) ([]model.Video, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var videos []model.Video
	if err := store.DB().Where("id IN ?", ids).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

// GetVideoByID 查询单个视频
func GetVideoByID(store DBProvider, id uint) (*model.Video, error) {
	var v model.Video
	err := store.DB().First(&v, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}
