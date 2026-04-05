# 统一请求日志与业务日志设计

> 日期：2026-04-05
> 主题：RequestLogMiddleware、zap 全局日志器、日志职责划分

---

## §1 问题背景

最近在项目里新增了统一请求日志中间件：

```go
h.Use(middleware.RequestLogMiddleware())
```

随后产生了两个常见问题：

1. 有了统一日志中间件，是否还需要在业务代码里单独打印日志？
2. 当前项目里的日志最终保存到了哪里？

这篇笔记专门回答这两个问题，并梳理当前项目的日志链路。

---

## §2 当前项目的统一日志是如何接入的

统一请求日志中间件是在服务启动时全局挂载的：

文件位置：
- `video-platform/main.go:42-44`

关键代码：

```go
h := server.Default(server.WithHostPorts(":" + port))
h.Use(middleware.RequestLogMiddleware())
h.Use(cors.Default())
```

这意味着每个 HTTP 请求都会先经过 `RequestLogMiddleware()`，再进入路由匹配、鉴权中间件和具体 Handler。

---

## §3 RequestLogMiddleware 做了什么

文件位置：
- `video-platform/pkg/middleware/request_log.go:20-49`

中间件完整流程如下：

```text
请求进入
    ↓
记录开始时间 start
    ↓
读取请求头 X-Request-Id
    ↓
若为空则生成新的 request_id
    ↓
写入 Hertz 上下文 c.Set("request_id", ...)
    ↓
写入响应头 X-Request-Id
    ↓
c.Next(ctx) 执行后续中间件和 Handler
    ↓
读取 path / user_id / status_code
    ↓
计算耗时 cost_ms
    ↓
输出一条统一请求日志
```

核心代码：

```go
func RequestLogMiddleware() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        start := time.Now()

        requestID := string(c.GetHeader(requestIDHeader))
        if requestID == "" {
            requestID = newRequestID()
        }

        c.Set(requestIDKey, requestID)
        c.Response.Header.Set(requestIDHeader, requestID)

        c.Next(ctx)

        path := c.FullPath()
        if path == "" {
            path = string(c.Path())
        }

        logger.L().Info("request completed",
            zap.String("request_id", requestID),
            zap.String("method", string(c.Method())),
            zap.String("path", path),
            zap.Uint("user_id", c.GetUint("user_id")),
            zap.Int("status_code", c.Response.StatusCode()),
            zap.Int64("cost_ms", time.Since(start).Milliseconds()),
        )
    }
}
```

---

## §4 为什么统一请求日志里能拿到 user_id

因为鉴权中间件会在 JWT 校验成功后，把用户信息写入 Hertz 上下文：

文件位置：
- `video-platform/pkg/middleware/auth.go:56-60`

关键代码：

```go
c.Set("user_id", claims.UserID)
c.Set("username", claims.Username)

c.Next(ctx)
```

执行顺序如下：

```text
RequestLogMiddleware
    ↓
AuthMiddleware（部分路由）
    ↓
业务 Handler
    ↓
回到 RequestLogMiddleware 输出日志
```

所以 `RequestLogMiddleware()` 在 `c.Next(ctx)` 返回之后，就能读取到后续中间件写入的 `user_id`。

结论：

- 已登录且鉴权成功的请求：日志中会有真实 `user_id`
- 未登录接口或鉴权失败请求：`user_id` 一般为 `0`

---

## §5 统一请求日志记录了哪些字段

当前请求日志字段包括：

| 字段 | 含义 |
|------|------|
| `request_id` | 请求链路 ID，用于串联上下游日志 |
| `method` | HTTP 方法，如 `GET`、`POST` |
| `path` | 路由路径或原始请求路径 |
| `user_id` | 当前登录用户 ID，未登录时通常为 `0` |
| `status_code` | HTTP 响应状态码 |
| `cost_ms` | 请求处理耗时，单位毫秒 |

这类日志属于“访问日志”或“链路日志”，重点是回答：

- 谁访问了哪个接口
- 请求是否成功
- 请求耗时如何
- 这次请求对应哪个 `request_id`

---

## §6 有了统一日志中间件，还需要单独日志吗

需要。统一日志中间件不能替代业务日志。

### 6.1 统一请求日志负责什么

统一请求日志负责记录“请求级”的公共信息：

- 请求入口
- 请求方法与路径
- 请求用户
- 状态码
- 耗时

它适合做接口访问统计、性能分析、链路追踪。

### 6.2 业务日志负责什么

业务日志负责记录“业务细节”和“异常原因”：

- 查询数据库失败
- Redis 缓存命中/未命中
- 文件上传失败
- 权限校验失败
- 事务执行失败

例如下面这种信息，统一请求日志是无法替代的：

```go
log.Printf("[视频模块][投稿] 保存视频文件失败 user_id=%d: %v", userID, err)
```

因为它回答的是：

- 到底哪一步失败了
- 失败原因是什么
- 关键业务参数是什么

而统一请求日志只会告诉你“这个请求最后返回了 500，用了 23ms”。

### 6.3 正确理解：两类日志互补

最合理的职责划分是：

- `RequestLogMiddleware()`：记录每个请求的一条总日志
- Handler / Service / DAO：在关键失败点打印业务错误日志

因此答案是：

**有了统一日志中间件以后，仍然需要单独日志。**

---

## §7 当前日志输出保存在哪里

当前项目没有把日志写入固定文件，而是输出到标准输出和标准错误。

文件位置：
- `video-platform/pkg/logger/logger.go:15-18`

关键配置：

```go
cfg := zap.NewProductionConfig()
cfg.Encoding = "json"
cfg.OutputPaths = []string{"stdout"}
cfg.ErrorOutputPaths = []string{"stderr"}
```

这说明当前日志落点是：

- 普通日志：`stdout`
- 错误输出：`stderr`

因此在不同运行方式下，日志位置会不同：

| 运行方式 | 日志去向 |
|----------|----------|
| `go run .` | 直接打印到当前终端 |
| `go run . > app.log 2>&1` | 被 shell 重定向到 `app.log` |
| Docker 容器运行 | 通过 `docker logs` 查看 |

结论：

**当前代码默认不会生成项目内固定日志文件。**

---

## §8 为什么项目里的 log.Printf 也能统一格式输出

项目在初始化 zap 日志器之后，还做了一层桥接，把标准库 `log` 的输出重定向到 zap：

文件位置：
- `video-platform/pkg/logger/logger.go:35-37`
- `video-platform/pkg/logger/logger.go:62-70`

关键代码：

```go
stdWriter := &zapStdWriter{logger: lg.Named("stdlog").WithOptions(zap.AddCallerSkip(1))}
log.SetFlags(0)
log.SetOutput(stdWriter)
```

桥接后的效果是：

- `logger.L().Info(...)` 会输出结构化 JSON 日志
- `log.Printf(...)` 也会被转成 zap 日志输出

所以项目虽然同时存在 `logger.L()` 和 `log.Printf()` 两种写法，但最终都走统一的日志出口。

---

## §9 示例：一次鉴权请求的日志链路

```text
客户端发起 POST /api/v1/video/publish
    ↓
RequestLogMiddleware 生成 request_id
    ↓
AuthMiddleware 校验 JWT 并写入 user_id
    ↓
PublishVideo Handler 执行业务逻辑
    ↓
若保存文件失败，Handler 内部打印业务错误日志
    ↓
RequestLogMiddleware 在请求结束后打印统一访问日志
```

这两条日志的角色不同：

1. 业务错误日志：告诉你“投稿失败，失败点在保存文件”
2. 请求访问日志：告诉你“该请求返回 500，耗时多少，request_id 是多少”

线上排障时通常需要同时结合两者看。

---

## §10 关键代码位置

- `video-platform/main.go:20-25`：初始化全局日志器
- `video-platform/main.go:42-44`：挂载统一请求日志中间件
- `video-platform/pkg/middleware/request_log.go:20-49`：请求日志中间件实现
- `video-platform/pkg/middleware/request_log.go:51-57`：`request_id` 生成逻辑
- `video-platform/pkg/middleware/auth.go:39-60`：JWT 校验并写入 `user_id`
- `video-platform/pkg/logger/logger.go:13-40`：zap 初始化与标准库日志桥接
- `video-platform/pkg/logger/logger.go:62-70`：`log.Printf` 转发实现

---

## §11 推荐阅读

- CloudWeGo Hertz Middleware 文档
- Uber zap 官方文档
- Go 标准库 `log` 文档
- OpenTelemetry Trace / Request ID 最佳实践

---

## §12 总结

当前项目的日志体系分成两层：

1. `RequestLogMiddleware()` 负责统一记录每个请求的访问日志。
2. 各业务模块中的 `log.Printf(...)` 或 `logger.L().Error(...)` 负责记录具体错误和业务上下文。

所以：

- **统一请求日志不能替代业务日志**
- **当前日志默认输出到 stdout/stderr，而不是某个固定文件**
- **`request_id` 是串联访问日志与业务日志的关键字段，后续可以继续围绕它做链路追踪增强**
