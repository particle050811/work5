package service

import (
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"video-platform/biz/dal"
	"video-platform/biz/dal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestStore(t *testing.T) (*dal.Store, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:%d?mode=memory&cache=shared", time.Now().UnixNano())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 sqlite 失败: %v", err)
	}

	if err := gdb.AutoMigrate(
		&model.User{},
		&model.Video{},
		&model.Comment{},
		&model.VideoLike{},
		&model.Follow{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}

	store := &dal.Store{}
	setUnexportedField(t, store, "db", gdb)
	return store, gdb
}

func setUnexportedField(t *testing.T, target interface{}, fieldName string, value interface{}) {
	t.Helper()

	rv := reflect.ValueOf(target).Elem().FieldByName(fieldName)
	if !rv.IsValid() {
		t.Fatalf("字段不存在: %s", fieldName)
	}

	reflect.NewAt(rv.Type(), unsafe.Pointer(rv.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func createTestUser(t *testing.T, gdb *gorm.DB, username string) *model.User {
	t.Helper()

	user := &model.User{
		Username: username,
		Password: "hashed-password",
	}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

func createTestVideo(t *testing.T, gdb *gorm.DB, userID uint, title string) *model.Video {
	t.Helper()

	video := &model.Video{
		UserID:      userID,
		Title:       title,
		Description: "test video",
		VideoURL:    "/storage/videos/test.mp4",
	}
	if err := gdb.Create(video).Error; err != nil {
		t.Fatalf("创建测试视频失败: %v", err)
	}
	return video
}
