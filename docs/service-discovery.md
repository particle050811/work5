# FanOne 服务注册与发现说明

最后更新：2026-04-06

## 1. 选型

项目当前统一使用 `etcd` 作为注册中心，服务通信使用 `Kitex`。

原因：

- 课程项目部署轻量
- 与 `Kitex` 集成简单
- 本地开发与 Docker 联调成本低

## 2. 服务名约定

当前固定服务名如下：

- `fanone.user`
- `fanone.video`
- `fanone.interaction`
- `fanone.chat`

`gateway` 当前作为 HTTP 入口，不注册为下游消费目标。

## 3. 端口约定

| 服务 | 协议 | 默认地址 |
| --- | --- | --- |
| `gateway` | HTTP | `:8888` |
| `user-service` | Kitex RPC | `0.0.0.0:9001` |
| `video-service` | Kitex RPC | `0.0.0.0:9002` |
| `interaction-service` | Kitex RPC | `0.0.0.0:9003` |
| `chat-service` | Kitex RPC | `0.0.0.0:9004` |
| `chat-service` | WebSocket/HTTP | `:8889` |
| `etcd` | client | `127.0.0.1:2379` |

## 4. 启动流程

### 4.1 Provider

领域服务启动时：

1. 读取 `ETCD_ENDPOINTS`
2. 创建 `etcd registry`
3. 创建 Kitex server
4. 以固定服务名注册到 etcd

代码模式已经落在各服务入口，例如：

- `services/user/main.go`
- `services/video/main.go`
- `services/interaction/main.go`
- `services/chat/main.go`

### 4.2 Consumer

`gateway` 启动时：

1. 读取 `ETCD_ENDPOINTS`
2. 创建 `etcd resolver`
3. 初始化长生命周期 Kitex client
4. 通过服务名路由到可用实例

当前已接入：

- `fanone.user`
- `fanone.video`
- `fanone.interaction`

## 5. 环境变量

联调时至少需要这些变量：

```bash
ETCD_ENDPOINTS=127.0.0.1:2379
DB_DSN=root:123456@tcp(127.0.0.1:3306)/fanone?charset=utf8mb4&parseTime=True&loc=Local
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0
JWT_SECRET=fanone-microservices-secret-key-2024
STORAGE_ROOT=./storage
GATEWAY_HTTP_ADDR=:8888
USER_RPC_ADDR=0.0.0.0:9001
VIDEO_RPC_ADDR=0.0.0.0:9002
INTERACTION_RPC_ADDR=0.0.0.0:9003
CHAT_RPC_ADDR=0.0.0.0:9004
CHAT_HTTP_ADDR=:8889
```

## 6. 本地联调

### 6.1 推荐顺序

1. 启动 `etcd`
2. 启动 `mysql`
3. 启动 `redis`
4. 启动 `user-service`
5. 启动 `video-service`
6. 启动 `interaction-service`
7. 启动 `chat-service`
8. 启动 `gateway`

### 6.2 脚本

仓库根目录已提供：

- `scripts/dev-up.sh`
- `scripts/dev-down.sh`
- `scripts/dev-status.sh`

它们负责：

- 拉起 `etcd/mysql/redis`
- 在本地后台启动五个服务
- 注入仓库根 `storage/` 作为统一上传目录
- 输出 PID 与日志文件位置

## 7. Docker 联调

仓库已新增 [docker-compose.micro.yml](/home/particle/2025-2/west2onlie_GoWeb/work5/deploy/docker-compose.micro.yml)。

该文件包含：

- `etcd`
- `mysql`
- `redis`
- `user-service`
- `video-service`
- `interaction-service`
- `chat-service`
- `gateway`

服务容器当前基于 `golang:1.25` 直接挂载仓库源码并执行各服务自己的 `go run .`，用于课程项目首版联调。仓库内已不再保留单体运行入口，后续可再替换成各服务独立 `Dockerfile`。

## 8. 故障处理约束

- etcd 不可用时，服务启动失败，避免伪成功
- `gateway` 不缓存固定下游 IP，只依赖服务名发现
- 非幂等写接口默认不配置自动重试
- 下游超时应该收敛为统一错误响应，避免无限阻塞

## 9. 当前限制

- 还没有补注册中心可视化观测
- 还没有补摘机、租约续期、重试策略的集成测试
- `chat-service` 当前只实现了最小 WebSocket echo 能力
