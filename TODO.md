# FanOne Lab5 TODO

最后更新：2026-04-06

本文档用于跟踪 `work5` 的开发任务，基线需求见 [work5-request.md](/home/particle/2025-2/west2onlie_GoWeb/work5/work5-request.md)。

当前已知现状：

- 当前主分支仍为 `master`，尚未切换为 `main`
- 当前仓库已有基础 CI：CodeQL、`golangci-lint`、`unit-test`
- 根目录尚未看到 `.dockerignore`、`.editorconfig`、`.gitattributes`
- 尚未看到独立 `config/` 目录与 `config.yaml`
- 当前项目仍是单体结构，尚未进入微服务拆分
- 当前项目尚未看到服务注册与发现基础设施
- 现有 `interaction.proto` 仍保留“本次作业只需实现一级评论”的旧注释，需要按 Lab5 升级
- 现有 `relation.proto` 仍未定义聊天相关 WebSocket / 历史消息能力

## P0：先完成能交付的 Lab5 核心需求

### 0. 微服务改造方案先落地

- [ ] 明确微服务拆分方案
  - 最低建议：
    - `gateway`
    - `user-service`
    - `video-service`
    - `interaction-service`
    - `chat-service`

- [ ] 确定 RPC 技术栈
  - 建议：对外 Hertz，对内 Kitex

- [ ] 确定服务注册与发现方案
  - 建议优先：`etcd`
  - 需要覆盖：
    - 服务注册
    - 服务发现
    - 本地开发启动方式
    - Docker 部署方式

- [ ] 输出微服务目录重构方案
  - 目标：
    - 网关层和服务层边界清晰
    - 公共 proto / config / pkg 不混乱

- [ ] 先补一版架构文档
  - 建议新增：
    - `docs/architecture-microservices.md`
    - `docs/service-discovery.md`

### 1. 更新 IDL，补齐 Lab5 协议

- [ ] 为用户模块补充 MFA 相关接口
  - 建议文件：[video-platform/api/video/v1/user.proto](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/api/video/v1/user.proto)
  - 最低应包含：
    - 获取 MFA 二维码
    - 绑定 MFA

- [ ] 为视频模块补充视频流接口
  - 建议文件：[video-platform/api/video/v1/video.proto](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/api/video/v1/video.proto)
  - 最低应明确：
    - 视频流读取方式
    - Range / 分片读取策略
    - 返回头设计

- [ ] 为互动模块补齐“评论回复 + 评论点赞”协议
  - 当前文件：[video-platform/api/video/v1/interaction.proto](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/api/video/v1/interaction.proto)
  - 当前问题：
    - 仍写着“本次作业只需实现一级评论”
    - 只有视频点赞，没有评论点赞
    - `PublishCommentRequest` 缺少 `comment_id` / `parent_id` 语义

- [ ] 为社交模块补充聊天协议
  - 当前文件：[video-platform/api/video/v1/relation.proto](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/api/video/v1/relation.proto)
  - 最低应明确：
    - 建立聊天连接
    - 消息结构
    - 历史消息拉取或离线补偿策略
    - AI 消息与普通用户消息如何区分

- [ ] 执行代码生成并同步更新 Swagger
  - 相关目录：
    - [video-platform/biz/model/api](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/model/api)
    - [video-platform/swagger](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/swagger)
  - 注意：
    - 带 `Code generated` 标记的文件只能通过生成命令更新

### 2. 实现 WebSocket 聊天主链路

- [ ] 设计聊天数据模型
  - 建议新增：
    - `conversation`
    - `message`
    - 可选 `conversation_member`
  - 建议目录：[video-platform/biz/dal/model](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/dal/model)

- [ ] 设计聊天持久化与实时投递方案
  - 要求：`Redis + MySQL`
  - 最低建议：
    - MySQL 存历史消息
    - Redis 做在线路由、消息分发、未读或会话缓存
    - 聊天服务独立为 `chat-service`

- [ ] 基于 Hertz WebSocket 实现聊天 handler
  - 建议目录：[video-platform/biz/handler/v1](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/handler/v1)
  - 建议内容：
    - 建链鉴权
    - 心跳保活
    - 用户上线/下线
    - 消息发送与广播

- [ ] 实现聊天 service
  - 建议目录：[video-platform/biz/service](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/service)
  - 最低应覆盖：
    - 创建会话
    - 保存消息
    - 投递消息
    - 离线消息或历史消息读取

- [ ] 增加聊天相关 e2e / 集成测试
  - 建议新增：[test/relation_ws.go](/home/particle/2025-2/west2onlie_GoWeb/work5/test/relation_ws.go)
  - 最低覆盖：
    - 两个用户建链
    - 互发消息
    - AI 插入回复

### 3. 实现 MFA

- [ ] 扩展用户表结构，增加 MFA 状态字段
  - 建议文件：
    - [video-platform/biz/dal/model/user.go](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/dal/model/user.go)
    - [video-platform/biz/dal/store.go](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/dal/store.go)

- [ ] 生成并返回 MFA 二维码
  - 建议落点：[video-platform/biz/service/user_service.go](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/service/user_service.go)

- [ ] 实现 MFA 绑定与校验逻辑
  - 最低应明确：
    - 绑定前提
    - 验证码校验
    - 绑定后的登录流程是否升级为二次校验

- [ ] 增加 MFA 测试
  - 建议新增：
    - 单测：`pkg` 或 `service`
    - e2e：`test/user.go`

### 4. 实现视频流

- [ ] 为视频文件提供流式读取能力
  - 建议文件：[video-platform/biz/handler/v1/video_handler.go](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/handler/v1/video_handler.go)

- [ ] 支持浏览器常见的 Range 请求
  - 目标：
    - 能够拖动进度条
    - 能正确返回 `206 Partial Content`

- [ ] 增加视频流测试
  - 建议覆盖：
    - 整文件读取
    - Range 分段读取
    - 非法范围

### 5. 补齐评论回复与评论点赞

- [ ] 扩展评论表结构，支持父子评论
  - 建议文件：[video-platform/biz/dal/model](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/dal/model)

- [ ] 扩展点赞模型，支持视频与评论两类目标
  - 要点：
    - 幂等
    - 目标类型区分
    - 计数更新一致性

- [ ] 重写互动模块测试
  - 当前文件：[test/interaction.go](/home/particle/2025-2/west2onlie_GoWeb/work5/test/interaction.go)
  - 最低应覆盖：
    - 对视频评论
    - 对评论回复
    - 对评论点赞
    - 删除他人评论失败

## P1：补齐工程要求，避免答辩失分

### 0. 微服务基础设施

- [ ] 为各服务拆分独立启动入口
  - 建议结构：
    - `cmd/gateway`
    - `cmd/user-service`
    - `cmd/video-service`
    - `cmd/interaction-service`
    - `cmd/chat-service`

- [ ] 为各服务拆分配置文件
  - 最低要求：
    - 服务级配置
    - 公共基础设施配置

- [ ] 为服务间调用定义公共 proto / kitex idl
  - 避免 HTTP DTO 直接充当内部 RPC 协议

- [ ] 接入服务注册与发现
  - 最低应验证：
    - 服务启动自动注册
    - 网关可动态发现下游服务
    - 下游实例变更后无需硬编码改地址

- [ ] 设计服务间超时、重试、降级策略
  - 尤其关注：
    - 网关 -> 用户服务
    - 网关 -> 视频服务
    - 网关 -> 互动服务
    - 网关 -> 聊天服务

### 1. 分支与 PR 流程

- [ ] 将主分支从 `master` 切换为 `main`
- [ ] 配置 GitHub 仓库保护，禁止直接推送主分支
- [ ] 后续变更改为 PR 合并
- [ ] 约定 PR 标题规范，并在文档中写明

### 2. 补齐仓库工程文件

- [ ] 新增 [.dockerignore](/home/particle/2025-2/west2onlie_GoWeb/work5/.dockerignore)
- [ ] 新增 [.editorconfig](/home/particle/2025-2/west2onlie_GoWeb/work5/.editorconfig)
- [ ] 新增 [.gitattributes](/home/particle/2025-2/west2onlie_GoWeb/work5/.gitattributes)
- [ ] 检查 [.gitignore](/home/particle/2025-2/west2onlie_GoWeb/work5/.gitignore) 是否足够覆盖构建产物、临时文件、上传文件

### 3. 配置治理

- [ ] 新增 `config/` 目录
  - 建议结构：
    - `config/config.yaml`
    - `config/sql/init.sql`
    - `config/services/*.yaml`

- [ ] 接入 Viper 并支持配置热更新
  - 最低要求：
    - 服务端能感知配置变更
    - 热更新日志明确可见

- [ ] 清理散落的硬编码配置
  - 重点关注：
    - JWT 配置
    - Redis 配置
    - 上传目录
    - 限流阈值
    - AI / MCP 配置

### 4. 参数校验与错误处理

- [ ] 为核心接口补齐参数校验
  - 重点文件：
    - [video-platform/api/video/v1](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/api/video/v1)
    - [video-platform/biz/handler/v1](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/biz/handler/v1)

- [ ] 统一错误处理链路
  - 目标：
    - 返回错误码稳定
    - 日志格式符合 AGENTS.md
    - 避免 handler / service 重复打日志

### 5. 流量治理

- [ ] 接入 Sentinel
  - 至少覆盖：
    - 视频流接口
    - 聊天连接
    - 登录相关接口
    - 网关到下游服务的关键入口

- [ ] 提供自定义治理配置
  - 包括：
    - 限流
    - 熔断或降级
    - 可观测日志

### 6. 代码复用与常量治理

- [ ] 清理重复分页逻辑
- [ ] 清理重复鉴权上下文读取逻辑
- [ ] 清理重复响应构造逻辑
- [ ] 抽取常量包
  - 至少管理：
    - action type
    - 业务状态值
    - Redis key 前缀
    - 错误文案
    - 默认分页参数

## P2：完善测试、CI 与文档

### 1. 单元测试

- [ ] 为新增聊天 service 增加单测
- [ ] 为 MFA 增加单测
- [ ] 为视频流边界场景增加单测
- [ ] 为评论回复 / 评论点赞增加单测
- [ ] 统计并记录测试覆盖率

### 2. CI 完善

- [ ] 检查 [golangci-lint workflow](/home/particle/2025-2/west2onlie_GoWeb/work5/.github/workflows/golangci-lint.yml) 是否只监听 `main`
- [ ] 检查 [unit-test workflow](/home/particle/2025-2/west2onlie_GoWeb/work5/.github/workflows/unit-test.yml) 是否覆盖 `video-platform` 与 `test`
- [ ] 检查 [codeql workflow](/home/particle/2025-2/west2onlie_GoWeb/work5/.github/workflows/codeql.yml) 的扫描路径是否正确
- [ ] 视情况增加构建检查，确保 proto 更新后能成功编译
- [ ] 新增微服务构建检查
  - 最低要求：
    - gateway 可编译
    - 各服务可编译
    - kitex 生成代码与仓库保持一致

### 3. README 与部署文档

- [ ] 精简 [video-platform/README.md](/home/particle/2025-2/west2onlie_GoWeb/work5/video-platform/README.md)
  - 目标：
    - README 保持项目概览
    - 细节下沉到 `docs/`

- [ ] 新增 `docs/` 目录
  - 建议拆分：
    - `docs/architecture.md`
    - `docs/architecture-microservices.md`
    - `docs/deploy.md`
    - `docs/chat-design.md`
    - `docs/cache-flow.md`
    - `docs/service-discovery.md`

- [ ] 补缓存流程图
  - 至少覆盖：
    - 热榜缓存
    - 聊天中 Redis 的使用路径

- [ ] 补部署文档
  - 最低包含：
    - 本地部署
    - Docker 部署
    - 服务器部署
    - 环境变量说明
    - 注册中心部署
    - 多服务启动顺序

- [ ] 在 README 中附上飞书报告链接

### 4. 报告材料

- [ ] 按 Lab5 要求编写答辩报告
  - 必含：
    - Problem Restatement
    - 问题解决
    - 单测覆盖率
    - 单元测试学习笔记
    - 单体 vs 微服务对比
    - 服务注册与发现设计说明

- [ ] 在报告中单独说明“代码复用性改动”

- [ ] 在报告中展示至少一组“优化前 / 优化后”
  - 可选主题：
    - 缓存
    - 数据库结构
    - 并发处理

## P3：可选但很加分

### 1. AI 聊天增强

- [ ] 为聊天接入 AI 自动回复
- [ ] 设计 AI 触发策略
- [ ] 接入 tool call
- [ ] 评估并接入 MCP
- [ ] 如接入福 uu tool，补充 jwch 授权接口

### 2. 异步化与性能优化

- [ ] 评估点赞、消息通知等是否可异步化
- [ ] 评估引入消息队列
- [ ] 为聊天系统补 Benchmark

### 3. 安全与治理增强

- [ ] 评估聊天链路的安全性
- [ ] 补充更多审计日志
- [ ] 预留可观测性扩展位
  - 可选：
    - Prometheus
    - Jaeger
    - 结构化日志

## 建议开发顺序

1. 先改 proto，统一协议与生成代码。
2. 先确定微服务拆分、Kitex、注册发现方案，再开始搬迁代码。
3. 再做聊天、MFA、视频流三个新增能力。
4. 同步补互动模块的评论回复和评论点赞，避免旧逻辑拖后腿。
5. 再补 `config/`、参数校验、Sentinel、工程文件。
6. 最后集中整理 README、部署文档、流程图和答辩报告。

## 每轮提交前检查

- [ ] `go test ./...` 通过
- [ ] `test/` e2e 通过
- [ ] `golangci-lint` 本地通过
- [ ] Swagger / proto 生成文件已同步
- [ ] README / docs / 报告同步更新
- [ ] PR 标题清晰可读
