# AGENTS.md

此文件为 Claude Code 和 Codex 在本仓库（Golang Lab5）工作的统一指南。

## 基本规范

- **默认使用中文** 回复用户与编写源码注释。
- **谨慎操作工作区**：若发现与当前任务无关的未提交改动，先确认来源后再行动，不要擅自回滚或覆盖。
- **避免破坏脚手架生成文件**：所有带有 "Code generated" 标记的文件只可通过对应的生成命令更新，禁止手改。
- **统一日志格式**：所有错误日志必须遵循 `[模块名][操作名] 错误描述 关键参数: %v` 的格式，便于快速定位问题。
- **curl 请求取消代理**：使用 curl 测试本地接口时，必须添加 `--noproxy localhost` 参数绕过代理，避免请求被代理拦截导致 502 错误。示例：`curl --noproxy localhost -s "http://localhost:8888/api/v1/video/list?user_id=14"`

### 日志格式规范

在 handler 层记录错误日志时，必须使用以下统一格式：

```go
log.Printf("[模块名][操作名] 错误描述 关键参数: %v", 参数值, err)
```

**格式说明：**
- `[模块名]`：用户模块、视频模块、互动模块、社交模块
- `[操作名]`：具体业务操作，如：注册、登录、点赞操作、发布评论等
- `错误描述`：简洁描述错误类型，如：查询用户失败、事务执行失败等
- `关键参数`：业务相关的关键 ID，如：`user_id=%d`、`video_id=%d`、`comment_id=%d` 等

**示例：**

```go
// ✅ 正确示例
log.Printf("[用户模块][登录] 查询用户失败 username=%s: %v", req.Username, err)
log.Printf("[互动模块][点赞操作] 查询目标视频失败 video_id=%d: %v", videoID, err)
log.Printf("[视频模块][投稿] 保存视频文件失败 user_id=%d: %v", userID, err)
log.Printf("[社交模块][关注操作] 查询目标用户失败 target_user_id=%d: %v", targetUserID, err)

// ❌ 错误示例
log.Printf("查询用户失败: %v", err)
log.Printf("点赞失败: %v", err)
log.Printf("保存视频失败: %v", err)
```

**优势：**
- 🔍 快速定位问题所属模块
- 🎯 明确具体失败的业务操作
- 📊 记录关键业务参数便于追踪
- ⚡ 提高线上问题排查效率

## 学习笔记整理规则

仅当用户明确提出“整理笔记”或“写入笔记”时才执行，目录结构如下：

- `learning/`
  - `00-index.md`：索引（含日期、主题及链接）
  - `01-auth-and-jwt.md`：认证 & 双 Token
  - `02-hertz-and-routing.md`：Hertz / Kratos 路由
  - `03-redis-cache.md`：Redis / 缓存应用
  - `04-video-domain.md`：视频/互动/社交业务
  - 继续按主题扩展

整理笔记时需注明日期、问题与解答、示例代码、文件与行号引用（如 `internal/user/service.go:42`）以及推荐阅读链接，并更新 `00-index.md`。

## 项目概述

本仓库目标是基于当前视频平台项目继续完成 **Golang Lab5** 要求。当前需求基线见 `work5-request.md`，统一接口文档见 https://doc.west2.online/ 。除保留现有用户、视频、互动、社交能力外，还需重点补齐 **微服务拆分、Kitex RPC、服务注册与发现**，以及 WebSocket 聊天、MFA、视频流、配置治理、CI、单测与工程规范。**所有接口协议需要继续以 Protobuf 描述并通过脚手架生成服务桩**，后续迭代也应保持项目结构清晰、可演进。

## 最低交付接口

| 模块 | 必须实现的接口 | 说明 |
| --- | --- | --- |
| 用户 | 注册、登录、获取用户信息、上传头像 | 需返回 access/refresh token，头像上传保存到本地目录 |
| 视频 | 投稿、发布列表、搜索视频、热门排行榜 | 搜索需满足所有条件；排行榜需引入 Redis 缓存 |
| 互动 | 点赞操作、点赞列表、评论、评论列表、删除评论 | 仅需对视频点赞/评论；删除时校验作者 |
| 社交 | 关注/取关、关注列表、粉丝列表、好友列表 | 好友=互相关注 |

接口入参、返回结构及错误码必须与官方文档一致，所有分页接口需正确处理 `page_num`、`page_size`。

## 强制性附加要求

1. **框架与脚手架**：必须使用现代 HTTP 框架（推荐 Hertz/ Kratos）并通过官方脚手架生成基础代码；**IDL 统一改为 Protobuf**。可以使用 `hz` 的 protobuf 流程或 Kratos + protobuf。
2. **数据库设计**：使用关系型数据库（推荐 MySQL + GORM），合理设计用户、视频、互动、社交相关表，并考虑唯一约束、关联、软删除等需求。
3. **缓存与排行榜**：视频热门排行榜必须经过 Redis，常见策略为：首次计算后写入缓存，后续请求读取缓存，必要时设置 TTL 及穿透/击穿防护。
4. **双 Token**：实现访问令牌（15 分钟）与刷新令牌（7 天），所有需要认证的接口校验访问令牌。刷新接口需同时校验刷新令牌并颁发新令牌。
5. **访问控制**：删除评论、修改视频、关注/取关等动作必须校验请求用户是否有权限。
6. **文件上传**：投稿接口只需支持单次上传，保存至 `storage/videos/`（或其他约定目录），记录文件元数据。
7. **Docker 化**：提供 `Dockerfile` 与运行说明，可一键构建并启动服务（环境变量包括 DB/Redis/JWT 等配置）。
8. **项目结构图**：在 README 或 `docs/` 中提供目录树，帮助答辩时快速说明架构。

## 建议技术栈

- **语言**：Go >= 1.21
- **HTTP 框架**：CloudWeGo Hertz（或 Kratos / 其他现代框架）
- **接口定义**：Protobuf + `hz`/`protoc`/`kratos tool`
- **数据库**：MySQL + GORM
- **缓存/排行榜**：Redis（Hot list、点赞计数等）
- **身份认证**：JWT（Access + Refresh）
- **对象存储**：本地文件系统起步，可预留 MinIO / OSS 扩展能力
- **构建与部署**：Go Modules、Docker、（可选）docker-compose

## 当前项目结构（微服务）

```
work5/
├── pkg/
│   ├── auth/
│   ├── logger/
│   ├── middleware/
│   ├── response/
│   ├── storage/
│   └── util/
├── idl/
│   ├── http/                         # 对外 HTTP Protobuf
│   └── rpc/                          # 内部 RPC Protobuf
├── docs/swagger/                     # OpenAPI 文档
├── services/
│   ├── gateway/
│   ├── user/
│   ├── video/
│   ├── interaction/
│   └── chat/
├── storage/                         # 运行态上传目录
│   ├── avatars/
│   └── videos/
├── gen/rpc/                          # Kitex 生成代码
├── scripts/                          # 微服务启动/停止/冒烟脚本
└── deploy/                           # docker-compose 等部署文件
```

如果切换到 Kratos/其他框架，请保持职责划分一致，并在 README 中同步最新结构图。

## 开发流程与命令建议

### hz 脚手架命令

1. **初始化项目**（首次使用）
   ```bash
   hz model --out_dir . --model_dir idl/http/gen --idl idl/http/user/v1/user.proto --proto_path=.
   ```

2. **更新/新增模块**（添加新 proto 文件时）
   ```bash
   hz model --out_dir . --model_dir idl/http/gen --idl idl/http/video/v1/video.proto --proto_path=.
   hz model --out_dir . --model_dir idl/http/gen --idl idl/http/interaction/v1/interaction.proto --proto_path=.
   hz model --out_dir . --model_dir idl/http/gen --idl idl/http/relation/v1/relation.proto --proto_path=.
   ```

3. **依赖管理**
   ```bash
   go mod tidy
   go mod download
   ```

4. **本地运行**
   ```bash
   go run .
   # 服务默认监听 :8888
   ```

5. **编译构建**
   ```bash
   go build -o fanone-video .
   ./fanone-video
   ```

6. **构建镜像**
   ```bash
   docker build -t fanone-video:latest .
   docker run --env-file .env -p 8080:8080 fanone-video:latest
   ```

### Swagger 文档生成

使用 `protoc-gen-http-swagger` 插件为各模块生成独立的 OpenAPI 文档：

```bash
# 安装插件（首次）
go install github.com/hertz-contrib/swagger-generate/protoc-gen-http-swagger@latest

# 生成/更新各模块文档
protoc --http-swagger_out=docs/swagger/user --proto_path=. idl/http/user/v1/user.proto
protoc --http-swagger_out=docs/swagger/video --proto_path=. idl/http/video/v1/video.proto
protoc --http-swagger_out=docs/swagger/interaction --proto_path=. idl/http/interaction/v1/interaction.proto
protoc --http-swagger_out=docs/swagger/relation --proto_path=. idl/http/relation/v1/relation.proto
```

启动服务后访问 Swagger UI：
- 用户模块: http://localhost:8888/swagger/user/index.html
- 视频模块: http://localhost:8888/swagger/video/index.html
- 互动模块: http://localhost:8888/swagger/interaction/index.html
- 社交模块: http://localhost:8888/swagger/relation/index.html

### 注意事项

- Proto 文件需要引入 `api.proto`（Hertz HTTP 注解定义）
- 注解使用 `(.api.xxx)` 格式（带点号前缀）避免包名冲突
- 多个 proto 文件分开 `hz update` 会导致 `Register` 函数重名，需手动重命名为 `RegisterUser`、`RegisterVideo` 等

## 领域特定约束

- **分页**：`page_num` 从 1 开始，`page_size` 默认 10、最大 50。响应需返回 `page_num`、`page_size`、`total`、`items`。
- **搜索**：视频搜索条件需全部满足（AND 关系），可按标题/标签/作者组合查询。
- **点赞**：同一用户重复点赞时需幂等处理；取消点赞后排行榜及缓存需同步更新。
- **评论**：仅支持对视频的一级评论；禁止删除他人评论。
- **社交关系**：关注、粉丝与好友列表需分页；好友=互相关注。
- **热点排行榜**：使用 Redis sorted set 或缓存列表，注意 TTL、穿透、雪崩防范，可通过互斥锁、预热、随机过期等方式实现。
- **日志与监控**：记录关键请求/响应日志，至少包含 request_id、用户 ID、处理耗时。为后续作业留出扩展位。

## 测试与调试建议

- 使用 Apifox / Postman 对照官方文档调试；`base_url` 默认 `http://localhost:8080`。
- 单元测试覆盖 service 与 repository；重点验证权限校验、分页逻辑、Redis 缓存读写。
- 引入 `test/` 目录编写简单的 e2e 测试客户端，依次跑完注册→投稿→点赞→评论→关注流程。
- 新功能上线前至少手动验证：
  1. 双 Token 登录、刷新；
  2. 投稿上传并能在列表 / 搜索 / 热榜中展示；
  3. 点赞、评论、关注等操作权限正确；
  4. Docker 镜像可启动并访问 /ping 健康检查。

### 强制性测试要求

**每次代码修改后必须执行 e2e 测试**：
1. 修改或新增接口实现后，同步更新 `test/` 目录下的测试文件
2. 运行 e2e 测试客户端验证所有相关功能
3. 确保所有测试用例通过后再提交代码

测试执行流程：
```bash
# 1. 启动服务（后台运行）
   ./scripts/dev-up.sh

# 2. 等待服务启动后运行测试
cd test && go run .

# 3. 检查测试结果，确保全部通过
```

测试文件结构：
- `test/main.go`：测试入口与流程编排
- `test/types.go`：响应类型定义
- `test/user.go`：用户模块测试
- `test/video.go`：视频模块测试
- `test/interaction.go`：互动模块测试
- `test/relation.go`：社交模块测试（待添加）

新增接口时需：
1. 在 `types.go` 中添加对应的响应类型
2. 在对应模块文件中添加测试函数
3. 在 `main.go` 中添加测试用例调用

## Bonus 参考方向

- 完成官方文档全部接口（投稿分片、评论回复、消息通知等）。
- 点赞与播放量结合 Redis + 定时任务，实现更真实的热榜。
- 自定义投稿接口以支持大文件分片上传（可结合 MinIO）。
- 实现 WebSocket 聊天或通知。
- 引入 ElasticSearch / OpenSearch 进行全文检索，或利用其记录结构化日志。

请在后续任务中持续更新本文件，以保证团队成员能够快速了解当前项目约束与工作流程。
