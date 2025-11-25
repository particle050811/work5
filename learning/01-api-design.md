# API 设计规范

## 2025-01-25: 文件上传接口的 Proto 定义

### 问题

在 proto 文件中定义文件上传接口时，是否需要用 `bytes` 类型来接收文件数据？

```protobuf
// ❌ 错误做法
message PublishVideoRequest {
  bytes data = 1;           // 视频原始二进制
  string filename = 2;
  string content_type = 3;
}
```

### 解答

**不需要**。HTTP 文件上传的标准方式是 `multipart/form-data`，Hertz 等框架直接从 `RequestContext` 获取文件，不依赖 proto 定义。

#### 原因

1. **HTTP 文件上传标准**：使用 `multipart/form-data` 编码，不是 protobuf 序列化
2. **框架原生支持**：Hertz/Gin 提供了 `FormFile()` 等方法直接获取上传文件
3. **内存问题**：用 `bytes` 传大文件会导致整个文件加载到内存

#### 正确做法

**Proto 定义**（只定义非文件字段）：

```protobuf
// ✅ 正确做法
message PublishVideoRequest {
  string title = 1;
  string description = 2;
  // 文件通过 multipart/form-data 上传，不在 proto 中定义
}
```

**Handler 实现**：

```go
func PublishVideo(ctx context.Context, c *app.RequestContext) {
    // 从 form-data 获取文件
    file, err := c.FormFile("video")
    if err != nil {
        // 处理错误
        return
    }

    // 获取其他表单字段
    title := c.PostForm("title")
    description := c.PostForm("description")

    // 保存文件到本地
    savePath := fmt.Sprintf("storage/videos/%s", file.Filename)
    if err := c.SaveUploadedFile(file, savePath); err != nil {
        // 处理错误
        return
    }

    // 数据库只存 URL/路径，不存文件内容
    video := &model.Video{
        VideoURL:    "/videos/" + file.Filename,
        Title:       title,
        Description: description,
    }
    // db.Create(video)
}
```

### 文件引用

- `video-platform/api/video/v1/video.proto:143-148` - PublishVideoRequest 定义
- `video-platform/api/video/v1/video.proto:132-134` - UploadAvatarRequest 定义

### 关键点

| 场景 | 存储内容 | 说明 |
|------|----------|------|
| 上传请求 | multipart/form-data | 框架自动解析，不需要 proto 定义 |
| 数据库存储 | URL/路径字符串 | 如 `/videos/xxx.mp4` |
| API 响应 | URL 字符串 | 客户端通过 URL 访问文件 |

### 推荐阅读

- [Hertz 文件上传文档](https://www.cloudwego.io/zh/docs/hertz/tutorials/basic-feature/file-upload/)
- [multipart/form-data 规范](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/POST)
