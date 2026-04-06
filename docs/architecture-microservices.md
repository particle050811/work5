# FanOne 微服务架构说明

最后更新：2026-04-06

## 1. 架构目标

当前仓库已经完成“删除单体运行入口、统一切到微服务启动”的第一阶段，当前目标是形成下面的职责边界：

- `gateway`：对外 HTTP 入口，统一鉴权、错误码映射、Swagger、聚合返回
- `user-service`：注册、登录、刷新令牌、用户信息、头像信息维护
- `video-service`：投稿、发布列表、搜索、热榜、视频评论列表查询
- `interaction-service`：点赞、评论、删除评论、关注、粉丝、好友
- `chat-service`：WebSocket 建链、聊天 RPC、后续消息存储与在线路由

首版明确采用：

- 对外 REST：`Hertz`
- 对内 RPC：`Kitex + Protobuf`
- 注册发现：`etcd`
- 聊天接入：`chat-service` 直接暴露 `/ws/chat`

## 2. 当前目录

```text
work5/
├── docs/
├── deploy/
├── gen/rpc/
├── idl/
│   ├── http/
│   └── rpc/
├── pkg/
├── scripts/
├── storage/
│   ├── avatars/
│   └── videos/
├── services/
│   ├── gateway/
│   ├── user/
│   ├── video/
│   ├── interaction/
│   └── chat/
└── go.work
```

目录职责如下：

- `services/`：微服务入口模块
- `pkg/`：认证、日志、中间件、响应、存储等横切能力
- `idl/http/`：对外 HTTP Protobuf 与生成模型
- `idl/rpc/`：Kitex 使用的内部 RPC 协议
- `gen/rpc/`：Kitex 生成代码
- `deploy/`：本地/容器化联调文件
- `docs/`：架构与运维文档
- `docs/swagger/`：网关挂载的 OpenAPI 文档
- `storage/`：运行态上传文件目录，统一放在仓库根

## 3. 运行拓扑

```text
Client
  |
  v
gateway (:8888, HTTP)
  |
  +--> fanone.user         (:9001, Kitex)
  +--> fanone.video        (:9002, Kitex)
  +--> fanone.interaction  (:9003, Kitex)
  |
chat-service
  |- :9004 Kitex
  |- :8889 WebSocket/HTTP
  |
  +--> interaction-service  查询 follow graph

infra:
- etcd   :2379
- MySQL  :3306
- Redis  :6379
```

## 4. 网关职责

`gateway` 只做四类事情：

1. 接收 HTTP 请求并完成参数校验
2. 校验 JWT、中间件日志、统一错误码转换
3. 调用下游 Kitex 服务
4. 组装客户端需要的 HTTP 返回结构

`gateway` 不直接连接业务 DAO，不直接写表，不承载领域规则。

## 5. 服务边界

### 5.1 user-service

- 管理 `users`
- 对外提供注册、登录、刷新、用户信息、头像更新 RPC
- 当前领域实现已下沉到 `services/user/internal`

### 5.2 video-service

- 管理 `videos` 及热榜缓存
- 提供投稿、列表、搜索、热榜、评论列表查询 RPC
- 文件元数据归属于视频域，文件本身统一落在仓库根 `storage/`

### 5.3 interaction-service

- 管理 `likes`、`comments`、`follows`
- 首版把 relation 域并入 interaction，先保证服务数量可控
- 对外提供点赞、评论、删除评论、关注、粉丝、好友 RPC

### 5.4 chat-service

- 直接对外暴露 `/ws/chat`
- 内部提供 `Ping` RPC 作为首版连通性占位接口
- 当前 WebSocket 能完成 echo，后续继续补消息持久化、会话管理、历史消息查询

## 6. 共享与隔离

首版允许多个服务共享同一个 MySQL 和 Redis 实例，但必须满足：

- 共享基础设施，不共享跨服务直查表权限
- 跨服务数据访问优先走 RPC
- `pkg/` 只能沉淀横切能力，不能继续堆放领域逻辑

当前各服务已具备自己的 `internal/service` 与 `internal/repository`，后续重点是继续把 `main.go` 中的入口适配逻辑下沉到 `internal/handler`，并补齐更清晰的配置与 DTO 分层。

## 7. 当前已落地内容

- 已建立 `go.work`
- 已新增 `idl/rpc/*` 与 `gen/rpc/*`
- 已建立 5 个服务模块
- `gateway` 已通过 etcd resolver 连接 `user-service`、`video-service`、`interaction-service`
- `chat-service` 已同时具备 RPC 端口和 WebSocket 端口
- 5 个服务模块当前均可独立 `go build ./...`

## 8. 下一步迁移重点

1. 继续把入口适配逻辑从 `main.go` 下沉到 `services/*/internal/handler`
2. 继续精简 `gateway` 中的 HTTP 聚合逻辑，拆出更清晰的 handler/pack
3. 补 `chat-service` 的消息模型、历史消息和准入校验
4. 改造现有 `test/`，形成微服务模式 e2e
5. 为各服务补独立配置与 Dockerfile
