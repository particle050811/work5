# FanOne 微服务拆分研究与落地计划

最后更新：2026-04-06

本文档用于回答当前 `work5` 仓库“如何从单体 Hertz 项目演进到 Lab5 要求的微服务架构”这一问题，目标不是空泛讨论，而是输出一份可以直接指导后续目录重构、IDL 调整、服务开发、联调和答辩说明的执行计划。

## 1. 当前现状与问题

当前仓库的主体实现位于 `shared/`，本质上仍是**单体应用**：

- 只有一个 HTTP 入口：`main.go + router_gen.go + router.go`
- 各模块虽然已经按 `handler / service / dal` 分层，但仍在同一进程内直接调用
- 用户、视频、互动、社交共用一套 DB / Redis 初始化入口：`biz/dal/store.go`
- 现有 Protobuf 主要服务于 HTTP 接口生成，还没有区分“外部 HTTP 协议”和“内部 RPC 协议”
- 现有 `test/` 的 e2e 测试也是面向单个 HTTP 服务，而不是网关 + 多下游服务

这会带来几个直接问题：

- 不满足 2025 版作业对微服务、Kitex、服务注册与发现的明确要求
- 后续接入 WebSocket 聊天、MFA、视频流后，单进程职责继续膨胀
- 模块边界虽然存在，但仍停留在代码目录层，缺少进程级边界
- 无法清晰说明“网关职责”和“领域服务职责”的区别，答辩时容易被问到只是“分层单体”

因此，后续目标不是推翻现有代码重写，而是**在保留现有 Hertz + Protobuf 资产的基础上，逐步演进为网关 + 多 Kitex 服务的结构**。

## 2. 拆分目标

本轮微服务改造的最低目标如下：

1. 对外 HTTP 入口统一收敛到 `gateway`
2. 对内服务调用统一使用 `Kitex + Protobuf`
3. 使用 `etcd` 作为服务注册与发现中心
4. 按领域拆分至少 5 个服务：
   - `gateway`
   - `user-service`
   - `video-service`
   - `interaction-service`
   - `chat-service`
5. 文档中能够说明：
   - 为什么这样拆
   - 如何注册与发现
   - 本地如何启动
   - Docker 如何启动
   - 从单体迁移到微服务的步骤和风险

## 3. 推荐总体架构

推荐采用“一个对外网关 + 四个领域服务 + 一个注册中心 + 公共存储”的结构。

```text
Client / Web / App
        |
        v
     gateway (Hertz)
        |
        |  Kitex RPC + etcd 服务发现
        |
        +-------------------+--------------------+--------------------+------------------+
        |                   |                    |                    |
        v                   v                    v                    v
 user-service         video-service      interaction-service      chat-service
        |                   |                    |                    |
        +---------+---------+---------+----------+--------------------+
                  |                   |
                  v                   v
               MySQL                Redis
```

### 3.1 为什么保留网关层

保留 `gateway` 而不是让每个服务直接对外暴露 HTTP，原因如下：

- 现有项目已经基于 Hertz 实现了 HTTP 协议、Swagger 和部分中间件，改造成本最低
- JWT 鉴权、请求日志、统一响应、参数校验更适合集中在网关处理
- 答辩时更容易清楚表达“对外 HTTP，对内 RPC”的分层
- 首版明确采用“`gateway` 负责 REST API，`chat-service` 直接对外暴露 `/ws/chat`”的方案，不在首版实现 WebSocket 网关代理

### 3.2 为什么内部使用 Kitex

选择 `Kitex` 而不是继续服务间 HTTP 调用，原因如下：

- `work5-request.md` 已明确建议对内优先使用 Kitex
- Kitex 与 CloudWeGo 体系兼容性好，和现有 Hertz 技术栈一致
- 能自然接入服务发现、超时、重试等治理能力
- 后续答辩时可以明确展示“外部接口协议”和“内部领域 RPC 协议”两层设计

### 3.3 为什么注册中心选择 etcd

`etcd` 作为本项目首选方案，原因如下：

- 依赖轻，适合课程项目本地启动和 Docker 部署
- 社区成熟，Kitex/Hertz 生态常见
- 足以覆盖本次作业要求的服务注册、服务发现、健康租约、实例上下线
- 相比 Nacos，部署和调试路径更短；相比 Consul，国内课程资料和示例通常更偏向 etcd

## 4. 服务边界设计

拆分服务时必须坚持“领域边界清晰，跨服务通过 RPC 交互，不把单体代码简单搬目录”。

### 4.1 gateway

职责：

- 对外提供 HTTP API
- 继续挂载 Swagger
- 统一鉴权、请求日志、trace/request_id、中间件
- 做 HTTP DTO 到 RPC DTO 的协议转换
- 做统一错误码映射和响应封装
- 聚合多服务结果

不应承担的职责：

- 不直接写业务表
- 不直接操作 GORM DAO
- 不承载核心业务规则
- 不实现聊天消息存储、点赞事务、视频搜索等领域逻辑

建议接口归属：

- `/api/v1/user/*` 由 gateway 转发到 `user-service`
- `/api/v1/video/*` 由 gateway 转发到 `video-service`
- `/api/v1/interaction/*` 由 gateway 转发到 `interaction-service`
- `/api/v1/relation/*` 首版由 `interaction-service` 承接；当关系域能力独立膨胀后再拆出 `relation-service`
- `gateway` 不负责 WebSocket 代理
- `/ws/chat` 由 `chat-service` 直接对外暴露，客户端通过 JWT 握手鉴权建立连接

客户端聚合规则：

- 面向客户端展示的聚合字段统一优先由 `gateway` 负责
- 领域服务只返回本领域稳定数据，不主动拼装其他服务的展示 DTO
- 领域内必要校验由服务自己发起下游 RPC，例如视频存在性校验、用户存在性校验

### 4.2 user-service

领域职责：

- 注册
- 登录
- 刷新 Token 所需用户信息校验
- 用户资料查询
- 头像上传元数据管理
- MFA 二维码生成与绑定

拥有的数据：

- `users`
- `user_mfa_secrets`，或者在 `users` 表中扩展 `mfa_enabled`、`mfa_secret`

对外提供的 RPC：

- `CreateUser`
- `VerifyUserPassword`
- `GetUserProfile`
- `UpdateAvatar`
- `CreateMFASecret`
- `BindMFA`
- `VerifyMFA`

注意：

- 首版明确由 `gateway` 负责签发和校验 `access token` / `refresh token`
- `user-service` 负责身份真实性校验、MFA 状态管理、MFA 验证和会话源数据提供
- 登录采用两阶段流程：
  1. `gateway -> user-service` 验证用户名密码
  2. 若用户未开启 MFA，则 `gateway` 直接签发 token
  3. 若用户已开启 MFA，则 `user-service` 返回 `mfa_challenge/session`，由 `gateway` 调用 `VerifyMFA` 成功后再签发 token
- `refresh token` 不采用纯无状态方案，首版至少保存 `session_id` 或 `refresh_token_jti` 到 Redis，用于刷新、吊销和登出失效控制
- JWT 密钥只由 `gateway` 持有，其他服务不直接签发 token

### 4.3 video-service

领域职责：

- 投稿
- 视频元数据存储
- 发布列表
- 搜索
- 热门排行榜读取
- 视频流元信息管理

拥有的数据：

- `videos`
- 视频标签/检索相关表
- 热榜缓存 key

对外提供的 RPC：

- `PublishVideo`
- `ListVideosByUser`
- `SearchVideos`
- `GetHotVideos`
- `GetVideoByID`
- `GetVideoStreamMeta`

注意：

- 视频文件本身仍保存在本地 `storage/videos/`
- 首版微服务仍可通过共享 volume 挂载相同存储目录
- 若后续演进到 MinIO，只需要替换 `video-service` 的文件访问实现

### 4.4 interaction-service

领域职责：

- 视频点赞
- 点赞列表
- 评论
- 评论列表
- 删除评论
- 评论回复
- 评论点赞
- 关注/取关、关注列表、粉丝列表、好友列表

为什么把社交关系先并入 `interaction-service`：

- 你当前 TODO 最低拆分只要求 5 个服务，并未强制单独保留 `relation-service`
- 关注关系和点赞评论同属“用户与内容/用户之间的互动”
- 可以避免初期拆得过碎，降低课程项目联调复杂度

拥有的数据：

- `likes`
- `comments`
- `follows`

对外提供的 RPC：

- `LikeVideo`
- `ListLikedVideos`
- `PublishComment`
- `ListComments`
- `DeleteComment`
- `ReplyComment`
- `LikeComment`
- `FollowUser`
- `ListFollowings`
- `ListFollowers`
- `ListFriends`

注意：

- 首版 `interaction-service` 只维护 `follow graph` 与互动行为，不负责聊天准入策略本身
- `chat-service` 只把“是否互相关注”等结果作为准入输入，真正的会话准入、黑名单、免打扰等策略由 `chat-service` 自己维护
- 如果后续出现黑名单、好友分组、聊天权限设置、消息免打扰等独立关系域能力，再拆出 `relation-service`
- 当前首版方案不建议一开始就拆 6 个以上服务，否则开发压力明显上升

### 4.5 chat-service

领域职责：

- WebSocket 建链
- 会话管理
- 单聊消息收发
- 历史消息查询
- 在线状态路由
- `AI` 消息协同预留扩展点，但不纳入首版验收范围

拥有的数据：

- `conversations`
- `conversation_members`
- `messages`
- Redis 在线状态、连接映射、最近会话缓存

对外提供的能力：

- WebSocket 接入
- `SendMessage`
- `ListMessages`
- `CreateConversation`
- `MarkConversationRead`

注意：

- `chat-service` 是最适合保持“服务内 HTTP/WebSocket + 内部可选 RPC”的服务
- 首版明确让 `chat-service` 直接对外监听 `ws` 端口，不经过 `gateway`
- 对外 REST 风格的历史消息查询仍走 `gateway -> chat-service`
- `chat-service` 同时暴露 `RPC` 端口和 `WebSocket/HTTP` 端口

## 5. 数据与存储边界

微服务改造后，不能再默认所有服务共享同一份 `biz/dal` 包直接互相访问数据。

### 5.1 可接受的首版方案

课程项目首版允许多个服务共享同一个 MySQL 实例和同一个 Redis 实例，但要注意：

- 共享的是**基础设施实例**，不是共享 DAO 代码
- 每个服务只操作自己拥有的表
- 其他服务如需相关数据，必须通过 RPC 获取，而不是跨服务直接查表

### 5.2 推荐表归属

| 表/资源 | 推荐归属服务 |
| --- | --- |
| `users` | `user-service` |
| `videos` | `video-service` |
| `likes` | `interaction-service` |
| `comments` | `interaction-service` |
| `follows` | `interaction-service` |
| `conversations` / `messages` | `chat-service` |
| `storage/avatars` | `user-service` 负责元数据 |
| `storage/videos` | `video-service` 负责元数据 |
| 热榜缓存 | `video-service` |
| 在线状态缓存 | `chat-service` |

### 5.3 跨服务数据访问约束

必须遵循以下规则：

1. `interaction-service` 需要校验视频是否存在时，调用 `video-service`
2. `video-service` 只返回作者 `user_id` 和视频元数据，作者昵称、头像等展示字段统一由 `gateway` 聚合
3. `chat-service` 如需查询“是否互相关注”等关系信息，调用 `interaction-service`
4. `chat-service` 内部维护真正的会话准入策略，不把聊天规则回塞给 `interaction-service`
5. `gateway` 不能直接连表替代领域服务

## 6. IDL 与代码生成规划

当前仓库已有 `shared/api/video/v1/*.proto`，这批文件更偏向 HTTP 接口协议。微服务改造后，建议拆成两层 IDL。

### 6.1 外部 HTTP IDL

保留现有风格，继续服务于 `hz` 或 Hertz HTTP 路由生成。

建议目录：

```text
idl/http/video/v1/
  common.proto
  user.proto
  video.proto
  interaction.proto
  relation.proto
  chat.proto
```

特点：

- 带 `api.proto` 的 HTTP 注解
- 面向客户端请求与响应
- 可直接生成 Swagger

### 6.2 内部 RPC IDL

新增一套 Kitex 专用 IDL。

建议目录：

```text
idl/rpc/user/v1/user.proto
idl/rpc/video/v1/video.proto
idl/rpc/interaction/v1/interaction.proto
idl/rpc/chat/v1/chat.proto
idl/rpc/common/v1/common.proto
```

特点：

- 不携带 HTTP 注解
- 面向服务间调用
- 请求响应按领域建模，不强行复用前端 DTO

生成产物建议：

- HTTP 代码生成产物统一放到 `gen/http/...`
- Kitex 代码生成产物统一放到 `gen/rpc/...`
- 各服务通过内部适配层引用生成代码，不直接把生成目录与业务目录混在一起

### 6.3 为什么不要强行一套 proto 同时兼容 HTTP 和 RPC

因为两类协议关注点不同：

- HTTP 更关注接口易读性、错误码、分页结构、Swagger 展示
- RPC 更关注内部字段完整性、服务边界和序列化效率

如果强行共用，常见问题是：

- 内部 RPC 为了迁就 HTTP 命名变得奇怪
- 一个字段改动会同时影响外部接口和内部调用
- 难以做领域级拆分

结论：**允许复用公共 message，但 HTTP IDL 与 RPC IDL 应分层管理。**

## 7. 推荐目录重构方案

建议在仓库根目录逐步调整为如下结构：

```text
work5/
├── AGENTS.md
├── TODO.md
├── plan.md
├── docs/
│   ├── architecture-microservices.md
│   └── service-discovery.md
├── deploy/
│   ├── docker-compose.local.yml
│   ├── docker-compose.micro.yml
│   └── etcd/
├── gen/
│   ├── http/
│   └── rpc/
├── idl/
│   ├── http/
│   │   └── video/v1/
│   └── rpc/
│       ├── common/v1/
│       ├── user/v1/
│       ├── video/v1/
│       ├── interaction/v1/
│       └── chat/v1/
├── pkg/
│   ├── xenv/
│   ├── xlog/
│   ├── xtrace/
│   ├── xerrs/
│   └── xmiddleware/
├── configs/
│   ├── gateway/
│   ├── user-service/
│   ├── video-service/
│   ├── interaction-service/
│   └── chat-service/
├── services/
│   ├── gateway/
│   │   ├── cmd/
│   │   ├── biz/
│   │   ├── rpc/
│   │   └── go.mod
│   ├── user-service/
│   ├── video-service/
│   ├── interaction-service/
│   └── chat-service/
├── go.work
├── test/
│   ├── e2e/
│   ├── integration/
│   └── smoke/
└── scripts/
```

### 7.1 各目录职责

- `docs/`：架构文档、注册发现文档、启动手册、迁移记录
- `deploy/`：docker compose、etcd、MySQL、Redis 等部署文件
- `gen/`：HTTP / RPC 代码生成产物
- `idl/`：统一维护 HTTP 与 RPC 协议
- `pkg/`：只保留真正跨服务复用、且不携带领域语义的公共包
- `configs/`：服务级配置模板
- `services/`：每个微服务独立模块
- `test/`：按 e2e、集成测试、冒烟测试拆开

### 7.2 依赖管理约束

- 仓库采用 `monorepo + 多 go.mod + go.work` 方式组织
- 根目录 `go.work` 统一管理 `services/*` 与公共模块，避免每个服务单独写大量 `replace`
- CI、代码生成、测试命令默认都从仓库根目录执行
- 服务间只通过 `idl/`、`gen/`、`pkg/` 共享公共代码，不允许直接跨服务引用业务实现

### 7.3 `pkg/` 中允许放什么

允许：

- 日志封装
- 配置加载
- request_id / trace_id 注入
- 公共错误码定义
- 通用中间件
- 通用 JWT 工具

不建议放：

- 用户领域 DAO
- 视频搜索逻辑
- 点赞业务逻辑
- 聊天消息路由逻辑

原则：`pkg/` 只能放“横切能力”，不能放“领域能力”。

## 8. 服务注册与发现设计

本项目推荐 `etcd` 作为统一注册中心。

端口模型统一如下：

- `gateway`：只暴露 `HTTP` 端口
- `user-service`：只暴露 `RPC` 端口
- `video-service`：只暴露 `RPC` 端口
- `interaction-service`：只暴露 `RPC` 端口
- `chat-service`：暴露 `RPC` 端口和 `WebSocket/HTTP` 端口

### 8.1 基本流程

服务启动时：

1. 读取自身配置，包括服务名、监听地址、注册中心地址
2. 建立到 etcd 的连接
3. 申请租约并注册实例，例如：
   - `services/user/{instance_id}`
   - `services/video/{instance_id}`
4. 定时续租
5. 优雅退出时撤销注册

服务消费时：

1. `gateway` 或其他服务在启动时创建长生命周期的 Kitex client
2. Kitex client 通过 etcd resolver 按服务名发现可用实例
3. 调用失败时根据超时、重试和熔断策略处理
4. 实例下线后，下次解析不再使用失效地址

约束：

- 不在每次请求里临时创建新的 Kitex client
- `gateway` 不缓存下游固定 IP，只依赖服务名发现

### 8.2 推荐服务名

- `fanone.gateway`
- `fanone.user`
- `fanone.video`
- `fanone.interaction`
- `fanone.chat`

不要混用目录名、端口名、随意字符串作为服务名。服务名必须稳定，便于配置和排障。

### 8.3 本地开发启动方式

本地开发需要至少支持以下模式：

#### 模式 A：单体兼容模式

用于快速开发已有功能。

- 继续允许当前 `shared/` 单体运行
- 用于未拆分完成前的过渡验证
- 当 `gateway + user-service + video-service + interaction-service` 四条主链路打通后，单体模式进入只读维护，不再承接新功能
- 聊天、MFA、视频流等新增能力只在微服务模式中继续开发

#### 模式 B：微服务联调模式

用于答辩前和主线开发。

启动顺序建议：

1. `etcd`
2. `mysql`
3. `redis`
4. `user-service`
5. `video-service`
6. `interaction-service`
7. `chat-service`
8. `gateway`

建议提供脚本：

- `scripts/dev-up.sh`
- `scripts/dev-down.sh`
- `scripts/dev-status.sh`

### 8.4 Docker 部署方式

建议使用 `docker-compose.micro.yml` 管理一套最小可运行环境：

- `etcd`
- `mysql`
- `redis`
- `gateway`
- `user-service`
- `video-service`
- `interaction-service`
- `chat-service`

要求：

- 所有服务通过环境变量读取 etcd 地址
- 所有服务启动后自动注册
- `gateway` 不硬编码下游地址
- `storage/avatars` 和 `storage/videos` 通过 volume 挂载
- `mysql`、`redis`、`etcd` 必须配置 `healthcheck`
- 服务启动时需要带有限重试或等待依赖 ready 的机制
- 需要单独的 `migrate/init` 步骤或容器，负责初始化表结构
- 首版文件存储仅保证单机/单副本可用；若扩展为多副本部署，需要切换到 MinIO / OSS 一类共享对象存储

### 8.5 故障处理策略

至少在文档中说明以下策略：

1. etcd 暂时不可用时，服务启动失败并打印明确日志
2. 已注册服务与 etcd 连接中断时，租约过期后实例自动摘除
3. `gateway` 调下游超时时返回统一错误码，不能无限等待
4. 下游短时抖动时允许有限重试，但禁止对非幂等写操作无脑重试

## 9. 配置治理方案

微服务落地后，配置必须拆成“公共基础设施配置”和“服务级业务配置”。

### 9.1 公共配置

建议放在：

- `configs/shared/*.yaml`
- 或通过环境变量统一注入

包括：

- MySQL 地址
- Redis 地址
- etcd 地址
- 日志级别
- 公共 JWT 密钥

### 9.2 服务级配置

例如：

- `configs/gateway/config.yaml`
- `configs/user-service/config.yaml`
- `configs/video-service/config.yaml`

包括：

- 服务监听端口
- 服务名
- 该服务用到的存储目录
- 下游依赖的超时配置
- 业务开关，例如热榜 TTL、WebSocket 心跳间隔

### 9.3 环境变量建议

统一约定命名，避免每个服务风格不一致：

- `APP_ENV`
- `SERVICE_NAME`
- `HTTP_ADDR`
- `RPC_ADDR`
- `ETCD_ENDPOINTS`
- `MYSQL_DSN`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `JWT_SECRET`

## 10. 迁移实施步骤

改造不能一步到位重写，建议按以下阶段推进。

### 阶段 0：文档和骨架先行

目标：

- 输出本文档
- 新增 `docs/architecture-microservices.md`
- 新增 `docs/service-discovery.md`
- 先把目录骨架搭出来

交付物：

- 微服务目录树
- 服务边界说明
- etcd 选型说明
- 本地启动方案

### 阶段 1：抽离公共层

目标：

- 把真正可复用的公共能力抽到新的 `pkg/`
- 把当前单体里和领域无关的日志、配置、中间件整理出来

具体动作：

1. 整理公共日志与 request_id 能力
2. 整理 JWT、配置加载、错误码映射
3. 落定 `go.work`、`gen/http`、`gen/rpc` 的目录和依赖管理规则
4. 避免继续扩张旧的 `shared/pkg/` 为“大杂烩”

### 阶段 2：先落 gateway + user-service

原因：

- 用户链路最清晰，依赖最少
- JWT、MFA、头像上传都集中在这里
- 最适合作为第一个拆分样板

具体动作：

1. 复制并改造现有用户 handler/service/dal 到 `user-service`
2. 为 `gateway` 建立 `user-service` 的 Kitex client
3. 明确两阶段登录与 MFA challenge 流程
4. 完成注册、登录、刷新、用户信息、头像上传链路打通
5. 接入 etcd 注册与发现

阶段验收：

- `gateway` 不再直接依赖用户 DAO
- 用户接口全部通过 RPC 访问 `user-service`
- `refresh token` 已具备服务端会话状态或 `jti` 失效控制

### 阶段 3：拆 video-service

具体动作：

1. 迁移视频投稿、列表、搜索、热榜能力
2. 把视频文件元数据和热榜缓存归位到 `video-service`
3. 增加视频流接口设计

阶段验收：

- 发布列表、搜索、热榜走 `gateway -> video-service`
- 热榜缓存仅由 `video-service` 维护

### 阶段 4：拆 interaction-service

具体动作：

1. 迁移点赞、评论、删除评论
2. 同时把关注关系并入
3. 增补评论回复、评论点赞

阶段验收：

- `likes/comments/follows` 不再从其他服务直接读写
- 对视频和用户的校验通过 RPC 获取

### 阶段 5：落 chat-service

具体动作：

1. 定义聊天模型和 RPC/HTTP 协议
2. 实现 WebSocket 建链、心跳、收发消息
3. 接入 Redis 在线路由和 MySQL 历史消息
4. 明确 `chat-service` 直连 `/ws/chat` 的客户端接入方式

阶段验收：

- 两个用户可完成建链和互发消息
- 服务重启后可读取历史消息
- 不依赖 `gateway` 做 WebSocket 代理

### 阶段 6：补齐部署与测试

具体动作：

1. 补 docker-compose
2. 改造 e2e 测试为微服务模式
3. 增加服务发现、超时、下线场景测试

阶段验收：

- `docker compose up` 后可完成主流程验证
- e2e 至少覆盖注册、投稿、点赞、评论、关注、聊天主流程

## 11. 测试策略

微服务改造后，测试不能只停留在单服务单元测试。

### 11.1 单元测试

每个服务都要覆盖：

- service 层业务逻辑
- repository / dao 层
- 缓存读写与降级逻辑

### 11.2 集成测试

需要新增：

- `gateway -> user-service`
- `gateway -> video-service`
- `gateway -> interaction-service`
- `chat-service + redis + mysql`
- `gateway` 下游超时与实例摘除场景
- RPC 契约兼容性检查

### 11.3 e2e 测试

建议把现有 `test/` 拆成以下结构：

```text
test/
├── e2e/
│   ├── main.go
│   ├── user.go
│   ├── video.go
│   ├── interaction.go
│   └── chat.go
├── fixtures/
└── helper/
```

必测主链路：

1. 用户注册、登录、刷新 token
2. 投稿视频、查询列表、搜索、热榜
3. 点赞、评论、删除评论、关注
4. WebSocket 聊天互发消息
5. 服务发现生效，`gateway` 不依赖固定下游地址
6. WebSocket 断线重连与历史消息补偿
7. 幂等写操作验证，例如重复点赞、重复关注、重复刷新

## 12. 风险与应对

### 12.1 风险：一次拆太多服务，开发失控

应对：

- 首版只拆 5 个服务
- 关注关系先并入 `interaction-service`
- 优先完成“能跑通”的服务链路，再补治理细节

### 12.2 风险：HTTP DTO 与 RPC DTO 混用，导致协议混乱

应对：

- 明确拆分 `idl/http` 与 `idl/rpc`
- 允许公共 message 复用，但不要强行单套 proto 兼容全部场景

### 12.3 风险：网关重新写太多逻辑，变成第二个单体

应对：

- 网关只做鉴权、路由、聚合、协议转换
- 所有领域逻辑必须下沉到具体服务

### 12.4 风险：多服务共用数据库后仍然跨表操作

应对：

- 明确表归属
- 跨服务只允许 RPC，不允许跨服务直接查表

### 12.5 风险：聊天链路复杂，拖慢整体交付

应对：

- 先完成历史消息 + 单聊 + 在线投递
- 群聊、已读回执、离线推送等能力放到后续迭代

### 12.6 风险：长期维护单体和微服务两套链路，迁移失焦

应对：

- 给单体模式设置退场条件
- 新增能力只进入微服务实现
- e2e 与部署脚本优先迁移到微服务链路

## 13. 建议近期任务拆解

如果要从今天开始推进，建议按下面的任务顺序执行：

1. 新建 `docs/architecture-microservices.md`，把第 3、4、7 节整理成正式架构文档
2. 新建 `docs/service-discovery.md`，把第 8 节整理成单独文档
3. 新建 `idl/http` 与 `idl/rpc` 目录骨架
4. 先抽 `gateway` 和 `user-service` 的项目骨架
5. 接入 etcd，并完成最小注册发现链路
6. 打通 `注册 -> 登录 -> 获取用户信息` 的 `gateway -> user-service` 全流程
7. 再迁移 `video-service` 与 `interaction-service`
8. 最后补 `chat-service`、Docker、e2e 和 README 结构图

## 14. 最终结论

对当前仓库，最稳妥且最符合 Lab5 要求的方案是：

- **对外继续使用 Hertz 作为 gateway**
- **对内使用 Kitex + Protobuf 进行 RPC 通信**
- **使用 etcd 做服务注册与发现**
- **先拆 5 个服务：gateway、user-service、video-service、interaction-service、chat-service**
- **社交关注能力首版并入 interaction-service，避免过度拆分**
- **通过“文档先行、骨架先行、user-service 先行”的方式渐进迁移**

这样做的优点是：

- 最大限度复用现有 Hertz 项目资产
- 能满足课程对微服务、Kitex、注册发现的明确要求
- 工程复杂度仍控制在课程项目可交付范围内
- 后续扩展 `relation-service`、消息队列、对象存储也有清晰演进路径
- 关键分叉决策已经提前定死，后续实现时不再反复摇摆
