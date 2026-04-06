package storage

import (
	"log"
	"os"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

const defaultStorageRoot = "./storage"

func root() string {
	if value := os.Getenv("STORAGE_ROOT"); value != "" {
		return value
	}
	return defaultStorageRoot
}

// AvatarDir 返回头像存储目录。
func AvatarDir() string {
	return filepath.Join(root(), "avatars")
}

// VideoDir 返回视频存储目录。
func VideoDir() string {
	return filepath.Join(root(), "videos")
}

// AvatarURL 返回头像公开访问路径。
func AvatarURL(filename string) string {
	return "/storage/avatars/" + filename
}

// VideoURL 返回视频公开访问路径。
func VideoURL(filename string) string {
	return "/storage/videos/" + filename
}

// BindStatic 挂载静态文件目录
func BindStatic(h *server.Hertz) {
	// 头像目录：/storage/avatars/<filename>
	avatarDir := AvatarDir()
	if err := os.MkdirAll(avatarDir, 0o755); err != nil {
		log.Printf("创建头像目录失败: %v", err)
	}
	h.StaticFS("/storage/avatars", &app.FS{
		Root:        avatarDir,
		PathRewrite: app.NewPathSlashesStripper(2),
	})

	// 视频目录：/storage/videos/<filename>
	videoDir := VideoDir()
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		log.Printf("创建视频目录失败: %v", err)
	}
	h.StaticFS("/storage/videos", &app.FS{
		Root:        videoDir,
		PathRewrite: app.NewPathSlashesStripper(2),
	})
}
