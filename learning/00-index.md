# FanOne 视频平台学习笔记索引

## 笔记列表

| 日期 | 主题 | 文件 |
|------|------|------|
| 2025-01-25 | API 设计规范 | [01-api-design.md](./01-api-design.md) |
| 2025-11-25 | Protobuf 设计与字段规范 | [02-protobuf-design.md](./02-protobuf-design.md) |
| 2025-11-25 | Hertz 框架与 hz 脚手架 | [03-hertz-and-hz.md](./03-hertz-and-hz.md) |
| 2025-12-07 | Go Context 用法详解 | [04-go-context.md](./04-go-context.md) |
| 2025-12-15 | 互动模块实现详解 | [05-interaction-module.md](./05-interaction-module.md) |
| 2025-12-16 | 认证与双 Token 机制 | [06-auth-and-jwt.md](./06-auth-and-jwt.md) |

### 06-auth-and-jwt.md 内容概要

| 章节 | 内容 |
|------|------|
| §1 | 问题背景（为何不用 req.UserId） |
| §2 | 两种获取用户 ID 的方式对比（来源、可信度、使用场景） |
| §3 | JWT 中间件实现详解（完整流程、核心代码） |
| §4 | JWT Claims 结构（自定义字段、生成时写入） |
| §5 | 双 Token 机制（Access/Refresh、有效期、刷新流程） |
| §6 | Hertz 上下文存取机制（Set/Get、与标准 Context 区别） |
| §7 | 关键代码位置 |
| §8 | 最佳实践 |
| §9 | 推荐阅读 |

### 05-interaction-module.md 内容概要

| 章节 | 内容 |
|------|------|
| §1 | 模块概览（5 个接口、认证要求） |
| §2 | 点赞模块设计（数据模型、事务处理、增量更新、幂等性） |
| §3 | 评论模块设计（发布、删除事务） |
| §4 | 权限校验对比（点赞 vs 删除评论） |
| §5 | Redis 热榜同步 |
| §6 | 数据流总结 |
| §7 | 关键代码位置 |
| §8 | 推荐阅读 |

### 04-go-context.md 内容概要

| 章节 | 内容 |
|------|------|
| §1 | Context 概述（ctx 与 c 的区别） |
| §2 | 核心功能（传值、超时、取消、截止时间） |
| §3 | WithValue 链式原理（链表结构、查找机制、遮蔽） |
| §4 | 实际场景：HTTP 请求链路追踪 |
| §5 | 常用方法速查表 |
| §6 | 最佳实践 |
| §7 | 在 Hertz 项目中的应用位置 |

### 02-protobuf-design.md 更新记录

| 日期 | 新增内容 |
|------|----------|
| 2025-11-25 | #7 双Token、#8 身份从Token获取、#9 Followings/Followers区别、#10 点赞命名、#11 评论模块归属、#12 删评论参数精简 |

### 03-hertz-and-hz.md 内容概要

| 章节 | 内容 |
|------|------|
| §1 | Hertz 简介 |
| §2 | hz 脚手架工具（安装、命令、目录结构） |
| §3 | api.proto 详解（HTTP 注解、参数绑定、校验） |
| §4 | 生成的 Handler 结构 |
| §5 | 多模块路由冲突解决方案 |
| §6 | 当前项目 API 路由表（19 个接口） |

## 待整理主题

- [x] 认证 & 双 Token (`06-auth-and-jwt.md`) ✅ 2025-12-16
- [ ] Redis / 缓存应用 (`07-redis-cache.md`)
- [ ] 社交模块 (`08-relation-module.md`)
