# Codex 目录结构调整建议

最后更新：2026-04-06

## 1. 文档目的

本文档用于给后续 Codex/Claude Code 调整仓库结构时提供统一参考。

目标不是一次性大改，而是：

1. 让当前仓库更贴近实验室正式项目的目录习惯
2. 保持当前微服务骨架不被破坏
3. 把“迁移中过渡目录”逐步收口
4. 降低后续新增服务、补测试、写文档时的理解成本

## 2. 当前判断

当前仓库的大方向是正确的，已经是“单仓微服务”结构：

- `services/` 承载服务入口
- `idl/rpc/` 承载内部 RPC 协议
- `gen/rpc/` 承载 Kitex 生成代码
- `deploy/`、`scripts/`、`docs/` 已独立出来

但目前仍有明显“迁移中”痕迹：

- 各服务虽然已补最小 `internal/` 骨架，但业务实现尚未下沉
- `idl/http/` 还未独立出来

本轮已完成：

- 已统一服务目录为 `services/gateway`、`services/user`、`services/video`、`services/interaction`、`services/chat`
- 已为各服务补齐最小 `internal/{handler,service,repository}` 骨架
- 已把运行态上传目录统一迁到仓库根 `storage/`
- 已清理服务源码目录中的历史二进制产物
- 已拆除 `shared/`，顶层形成 `pkg/`、`idl/http/`、`docs/swagger/`

## 3. 总体建议

建议后续逐步收敛到下面这套结构：

```text
work5/
├── README.md
├── docs/
├── deploy/
├── scripts/
├── storage/
│   ├── avatars/
│   └── videos/
├── idl/
│   ├── http/
│   └── rpc/
├── kitex_gen/                 # 或保留 gen/rpc/kitex_gen，二选一
├── pkg/
├── services/
│   ├── gateway/
│   │   ├── main.go
│   │   └── internal/
│   ├── user/
│   │   ├── main.go
│   │   └── internal/
│   ├── video/
│   │   ├── main.go
│   │   └── internal/
│   ├── interaction/
│   │   ├── main.go
│   │   └── internal/
│   └── chat/
│       ├── main.go
│       └── internal/
├── test/
└── go.work
```

说明：

- 如果继续保留 `gen/rpc/kitex_gen`，也可以，不是必须改
- 如果目标是尽量贴近实验室正式项目习惯，优先考虑顶层 `kitex_gen/`
- `pkg/` 只保留横切公共能力，不再承载具体业务实现

## 4. 需要调整的重点

### 4.1 统一服务目录命名

当前目录：

- `services/gateway`
- `services/user`
- `services/video`
- `services/interaction`
- `services/chat`

当前状态：

- 已统一为短目录名风格
- 服务注册名、进程名仍保留 `user-service` 等语义化命名

建议：

方案 A，推荐：

```text
services/gateway
services/user
services/video
services/interaction
services/chat
```

原因：

- 更短
- 更贴近目录命名习惯
- 服务注册名、容器名、二进制名可以继续单独使用 `fanone.user`、`user-service`

方案 B：

```text
services/gateway-service
services/user
services/video
services/interaction
services/chat
```

不推荐原因：

- 目录名偏长
- `gateway-service` 可读性不如 `gateway`

### 4.2 给每个服务补齐 `internal/`

当前各服务已补最小 `internal/` 骨架，业务实现已开始下沉到 `internal/service` 与 `internal/repository`，但入口适配仍有一部分留在 `main.go`。

建议后续统一落成：

```text
services/user/
├── main.go
└── internal/
    ├── handler/
    ├── service/
    ├── repository/
    ├── pack/
    └── config/
```

说明：

- `handler/`：RPC 或 HTTP 入口适配层
- `service/`：领域逻辑
- `repository/`：数据访问
- `pack/`：DTO/响应转换
- `config/`：服务级配置读取与默认值

如果目录过多，可以先最小化为：

```text
internal/
├── handler/
├── service/
└── repository/
```

### 4.3 保持职责分离

当前已经完成第一轮拆分：

- 公共横切能力位于顶层 `pkg/*`
- HTTP 协议位于 `idl/http/*`
- Swagger 文档位于 `docs/swagger/*`
- 业务实现开始下沉到 `services/*/internal/*`

后续重点不是再引入新的共享总仓，而是继续缩小顶层公共层，只保留真正横切的能力。

### 4.4 独立 HTTP IDL

当前对外 HTTP proto 已迁到 `idl/http/*`。

这会带来两个问题：

1. 需要继续保持 HTTP 协议与业务实现分离
2. 需要避免在新的目录下再次堆叠“过渡层”

建议目标：

```text
idl/http/user/v1/
idl/http/video/v1/
idl/http/interaction/v1/
idl/http/relation/v1/
idl/rpc/user/v1/
idl/rpc/video/v1/
idl/rpc/interaction/v1/
idl/rpc/chat/v1/
```

这样 HTTP 与 RPC 协议会形成对称结构，理解成本最低。

### 4.5 处理上传目录

当前上传目录已迁移到仓库根：

- `storage/avatars`
- `storage/videos`

当前已通过 `STORAGE_ROOT` 统一注入运行态目录，不再把上传文件挂在服务源码目录下。

建议迁移为：

```text
storage/avatars/
storage/videos/
```

或者完全通过环境变量指定：

```bash
STORAGE_ROOT=./storage
AVATAR_DIR=./storage/avatars
VIDEO_DIR=./storage/videos
```

### 4.6 清理编译产物位置

当前历史二进制产物已从服务源码目录移除。

建议统一输出到：

```text
bin/
```

或：

```text
.runtime/bin/
```

同时补充 `.gitignore`。

### 4.7 脚本只放到 `scripts/`

当前 `gen/rpc/script` 也有脚本目录。

建议统一原则：

- `scripts/` 只放脚本
- `gen/` 或 `kitex_gen/` 只放生成产物

这样生成目录就不会混入人工维护脚本。

## 5. 哪些内容暂时不要动

以下内容虽然未来可以继续优化，但当前不建议优先改：

### 5.1 `deploy/`

当前 `deploy/docker-compose.micro.yml` 命名已经足够清晰，可继续保留。

### 5.2 `docs/`

当前已有：

- 微服务架构说明
- 服务发现说明

文档方向是对的，后续只需补根 README 做入口索引。

### 5.3 `test/`

当前顶层 `test/` 作为 e2e 测试目录是合理的，符合课程项目需求，不建议拆散。

## 6. 推荐迁移顺序

建议按下面顺序推进，避免一次性大范围改动：

1. 统一 `services/` 目录命名
2. 为每个服务补 `internal/` 目录骨架
3. 把运行态 `storage/` 提到仓库根目录
4. 把编译产物移出服务源码目录
5. 继续把入口适配逻辑下沉到 `services/*/internal/handler`
6. 继续把残余业务逻辑从 `main.go` 下沉到各服务
7. 补 `idl/http/`，再迁移现有 HTTP proto
8. 最后再决定是否保留 `gen/rpc/` 还是切顶层 `kitex_gen/`

## 7. 对 Codex 的执行约束

后续如果 Codex 要做目录改造，建议遵守以下约束：

1. 不一次性做“大重命名 + 大迁移 + 业务改造”
2. 每轮只做一个层面的调整，例如“只统一命名”或“只迁移 pkg”
3. 任何 `Code generated` 文件都必须通过生成命令更新，不能手改
4. 每次改动后同步修复 `go.work`、`go.mod`、导入路径和脚本
5. 每次目录改造后都执行 e2e 测试
6. 文档和目录树必须同步更新

## 8. 最终建议结论

当前仓库不是结构错误，而是“微服务迁移过渡态”。

后续调整时应坚持这三个方向：

1. 服务目录更统一
2. 共享代码更克制
3. 协议、生成物、运行态数据各归各位

如果要尽量贴近实验室正式项目习惯，建议优先靠拢：

- `pkg/`
- `idl/`
- `kitex_gen/`
- `services/*/internal`

而不是再造新的“大而全”共享目录。
