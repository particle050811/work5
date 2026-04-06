# Protobuf 设计与字段规范

## 2025-11-25: Proto 字段设计与文件拆分

### 1. 常见字段含义

#### `avatar_url` - 用户头像

用户头像的 URL 地址，用于前端展示用户头像图片。

```protobuf
message User {
  string avatar_url = 3; // 用户头像 URL，用于前端展示
}
```

#### `cover_url` - 视频封面

视频封面图的 URL，在列表页展示的缩略图。

```protobuf
message Video {
  string cover_url = 4;    // 视频封面图 URL，列表页展示的缩略图
  string description = 6;  // 视频描述/简介
}
```

---

### 2. 软删除设计：为何只用 `deleted_at`

#### 问题

软删除为什么只有 `deleted_at` 时间字段，而没有 `is_deleted` 布尔字段？

#### 解答

这是 **GORM 的标准软删除模式**，判断逻辑如下：

| `deleted_at` 值 | 状态 |
|----------------|------|
| `NULL` (空)    | 未删除 |
| 有时间戳       | 已删除 |

#### 不需要额外布尔字段的原因

1. **信息冗余**：时间字段本身就能表达"是否删除"
2. **额外价值**：`deleted_at` 还记录了删除时间，方便审计、数据恢复
3. **GORM 内置支持**：

```go
type User struct {
    ID        uint
    DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除字段
}

// GORM 自动处理：
db.Delete(&user)           // UPDATE users SET deleted_at='2025-01-01' WHERE id=1
db.Find(&users)            // SELECT * FROM users WHERE deleted_at IS NULL
db.Unscoped().Find(&users) // SELECT * FROM users (包含已删除)
```

4. **索引友好**：`WHERE deleted_at IS NULL` 在某些场景下更高效

---

### 3. 评论的 `video_id` 与 `parent_id` 关系

#### 问题

Comment 中的 `video_id` 和 `parent_id` 是只有一个有值吗？

#### 解答

不是，它们的关系是：

| 字段 | 说明 |
|------|------|
| `video_id` | **始终有值**，评论所属的视频 ID |
| `parent_id` | **可选**，父评论 ID（用于回复/嵌套评论） |

使用场景：
- **一级评论**：`video_id` 有值，`parent_id` 为空
- **回复评论**：`video_id` 有值，`parent_id` 指向被回复的评论

> 注：根据 AGENTS.md 要求，当前项目只需实现一级评论，`parent_id` 为扩展预留。

---

### 4. 嵌套响应结构 vs 扁平结构

#### 问题

为什么使用 `resp.Base.Code` 而不是常见的 `resp.Code`？

```protobuf
message RegisterResponse {
  BaseResponse base = 1;  // 嵌套方式
}
```

#### 对比

| 方式 | 访问路径 | 特点 |
|------|---------|------|
| 嵌套（当前） | `resp.Base.Code` | 复用 BaseResponse，改动集中 |
| 扁平（常见） | `resp.Code` | 访问直接，但每个响应重复字段 |

#### 嵌套方式的好处

- 如果要给 BaseResponse 加字段（如 `request_id`），只需改一处
- 语义上明确区分"通用状态"和"业务数据"

> 本项目采用嵌套方式是作业格式要求。

---

### 5. Proto 文件拆分

#### 问题

所有接口放一个文件太长，不方便维护。

#### 解决方案

按模块拆分为 5 个文件：

```
api/video/v1/
├── common.proto      # BaseResponse, PageParams
├── user.proto        # User, UserService (import common)
├── video.proto       # Video, VideoService (import common)
├── interaction.proto # Comment, InteractionService (import common, video)
└── relation.proto    # SocialProfile, RelationService (import common)
```

#### 依赖关系

```
common.proto ← user.proto
             ← video.proto ← interaction.proto
             ← relation.proto
```

#### 生成代码

```bash
hz update --idl api/video/v1/*.proto
```

---

### 文件引用

- `shared/api/video/v1/common.proto` - 通用类型定义
- `shared/api/video/v1/user.proto` - 用户模块
- `shared/api/video/v1/video.proto` - 视频模块
- `shared/api/video/v1/interaction.proto` - 互动模块
- `shared/api/video/v1/relation.proto` - 社交模块

### 推荐阅读

- [GORM 软删除文档](https://gorm.io/zh_CN/docs/delete.html#%E8%BD%AF%E5%88%A0%E9%99%A4)
- [Protobuf Style Guide](https://protobuf.dev/programming-guides/style/)

---

### 6. 为什么 Protobuf 不定义文件上传字段

#### 问题

视频投稿接口，为什么不在 proto 里直接定义 `bytes data = 3` 来传视频文件？

```protobuf
// 为什么不这样写？
message PublishVideoRequest {
  string title = 1;
  string description = 2;
  bytes data = 3;  // 视频文件
}
```

#### 解答

**Protobuf 不适合传输大型二进制文件**，原因如下：

| 问题 | 说明 |
|------|------|
| 内存占用 | protobuf 消息需要完全加载到内存才能解析，视频可能几十 MB 甚至 GB |
| 无法流式传输 | 必须等整个消息接收完才能处理 |
| 序列化开销 | 大文件的 base64 编码会增加约 33% 体积 |

#### 正确做法

使用 **HTTP multipart/form-data** 上传文件：

```protobuf
// proto 只定义元数据
message PublishVideoRequest {
  string title = 1;
  string description = 2;
}
```

```go
// handler 中分别处理
func PublishVideo(ctx context.Context, c *app.RequestContext) {
    // 1. 获取表单字段（绑定到 proto 生成的结构）
    title := c.FormValue("title")
    description := c.FormValue("description")

    // 2. 获取文件（流式处理，内存占用小）
    file, err := c.FormFile("data")
    if err != nil {
        // 处理错误
    }

    // 3. 保存文件
    c.SaveUploadedFile(file, "./storage/videos/"+file.Filename)
}
```

#### multipart/form-data 的优势

1. **流式传输**：边接收边写入磁盘，不占用大量内存
2. **浏览器原生支持**：`<input type="file">` 直接上传
3. **框架优化**：Hertz/Gin 的 `FormFile()` 专门处理文件流
4. **支持多文件**：可同时上传视频和封面图

#### 官方文档要求

根据 https://doc.west2.online/api-141535772.md：

| 字段 | 类型 | 说明 |
|------|------|------|
| data | binary | 视频原始数据（multipart/form-data） |
| title | string | 视频标题 |
| description | string | 描述 |

---

### 7. 登录响应需要双 Token

#### 问题

`LoginResponse` 只返回 `User` 数据，没有 token 字段。

#### 解答

根据作业要求（双 Token 认证），登录成功必须返回两个令牌：

```protobuf
message LoginResponse {
  BaseResponse base = 1;
  User data = 2;
  string access_token = 3;  // 访问令牌，有效期 15 分钟
  string refresh_token = 4; // 刷新令牌，有效期 7 天
}
```

同时需要添加刷新接口：

```protobuf
message RefreshTokenRequest {
  string refresh_token = 1;
}

message RefreshTokenResponse {
  BaseResponse base = 1;
  string access_token = 2;
  string refresh_token = 3; // 滑动刷新
}
```

#### 文件引用

- `shared/api/video/v1/user.proto:35-40`

---

### 8. UploadAvatarRequest 为何不需要 user_id

#### 问题

上传头像为什么不传 `user_id`？

```protobuf
message UploadAvatarRequest {} // 为空
```

#### 解答

**安全设计原则**：用户身份从 JWT Token 获取，而非客户端传递。

```
请求头: Authorization: Bearer <access_token>
请求体: multipart/form-data (avatar 文件)
          │
          ▼
    Auth 中间件解析 Token → user_id
          │
          ▼
    Handler 从 ctx 获取 user_id（不可伪造）
```

如果让客户端传 `user_id`：
- 用户 A 可以传 `user_id=B`，篡改用户 B 的头像（**安全漏洞**）

```go
func UploadAvatar(ctx context.Context, c *app.RequestContext) {
    userID := c.GetString("user_id")  // 从 token 解析，不可伪造
    file, _ := c.FormFile("avatar")
    // ...
}
```

---

### 9. Followings vs Followers 区别

#### 问题

`ListFollowingsRequest` 和 `ListFollowersRequest` 分不清。

#### 解答

| 接口 | 含义 | 例子 |
|------|------|------|
| `ListFollowings` | **关注列表**：我关注了谁 | 我 ──关注──▶ [张三, 李四] |
| `ListFollowers` | **粉丝列表**：谁关注了我 | [路人A, 路人B] ──关注──▶ 我 |

**记忆技巧**：
- Follow**ings** = 正在关注**的人**（主动）
- Follow**ers** = 关注**者**/粉丝（被动）

```protobuf
// 关注列表：我关注了谁
// user_id=A → 返回 A 关注的所有人
message ListFollowingsRequest {
  string user_id = 1;
  PageParams page = 2;
}

// 粉丝列表：谁关注了我
// user_id=A → 返回关注 A 的所有人
message ListFollowersRequest {
  string user_id = 1;
  PageParams page = 2;
}

// 好友列表：互相关注（只能查自己的）
message ListFriendsRequest {
  PageParams page = 1;  // 无 user_id，从 token 获取
}
```

#### 好友的定义

**好友 = 互相关注**（来源：统一接口文档 `https://doc.west2.online/`）

```
我 ◀──关注──▶ 张三  → 张三是我的好友
```

#### 文件引用

- `shared/api/video/v1/relation.proto:40-76`

---

### 10. 点赞接口命名：为未来扩展预留

#### 问题

`LikeActionRequest` 只有 `video_id`，未来扩展评论点赞时会混淆。

#### 解决方案

改名为 `VideoLikeActionRequest`，未来扩展时添加 `CommentLikeActionRequest`：

```protobuf
// 视频点赞（当前实现）
message VideoLikeActionRequest {
  string video_id = 1;
  LikeActionType action_type = 2;
}

// 评论点赞（未来扩展）
// message CommentLikeActionRequest {
//   string comment_id = 1;
//   LikeActionType action_type = 2;
// }
```

#### 文件引用

- `shared/api/video/v1/interaction.proto:41-50`

---

### 11. 评论列表接口的模块归属

#### 问题

`ListCommentsRequest` 按 `video_id` 查询，放在 `interaction.proto` 不合适。

#### 解答

应按**查询主体**划分模块：

| 查询主体 | 接口 | 所属模块 |
|----------|------|----------|
| 某视频的评论 | `ListVideoComments` | `video.proto` |
| 某用户的评论 | `ListUserComments` | `interaction.proto` |

修改后：

```protobuf
// video.proto - 查看视频的评论
message ListVideoCommentsRequest {
  string video_id = 1;
  PageParams page = 2;
}

// interaction.proto - 查看用户发表的评论
message ListUserCommentsRequest {
  string user_id = 1;
  PageParams page = 2;
}
```

#### 文件引用

- `shared/api/video/v1/video.proto:69-95`
- `shared/api/video/v1/interaction.proto:71-81`

---

### 12. DeleteCommentRequest 不需要 video_id

#### 问题

```protobuf
message DeleteCommentRequest {
  string video_id = 1;   // 冗余！
  string comment_id = 2;
}
```

#### 解答

`video_id` 是冗余字段，只需 `comment_id`：

```protobuf
message DeleteCommentRequest {
  string comment_id = 1;  // 数据库已记录视频归属
}
```

原因：
- 评论的 `video_id` 在数据库中已经记录
- 删除时只需 `comment_id` 定位记录
- Handler 需校验当前用户是否为评论作者

#### 文件引用

- `shared/api/video/v1/interaction.proto:83-88`

---

### 文件引用

- `shared/api/video/v1/video.proto:35-38` - PublishVideoRequest 定义
- `shared/api/video/v1/user.proto` - 用户模块（含双 Token）
- `shared/api/video/v1/relation.proto` - 社交模块（Followings/Followers/Friends）
- `shared/api/video/v1/interaction.proto` - 互动模块（点赞/评论）
