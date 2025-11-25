# FanOne 视频平台 API 文档

## 参考文档
- 官方文档 [README.md](https://doc.west2.online/doc-3579384.md)

## 最低要求 API (共 17 个)

### 用户模块 (4 个)
- [x] [注册](https://doc.west2.online/api-141535768.md)
- [x] [登录](https://doc.west2.online/api-141535769.md)
- [x] [用户信息](https://doc.west2.online/api-141535770.md)
- [x] [上传头像](https://doc.west2.online/api-141541887.md): 对当前用户上传头像

### 视频模块 (4 个)
- [x] [投稿](https://doc.west2.online/api-141535772.md): 使用 HTTP 单文件上传视频
- [x] [发布列表](https://doc.west2.online/api-141535773.md): 根据 user_id 查看指定人的发布列表
- [x] [搜索视频](https://doc.west2.online/api-141546426.md): 搜索指定关键字的视频
- [x] [热门排行榜](https://doc.west2.online/api-141545896.md): 根据点击量获取排行榜，要求使用 Redis 缓存

### 互动模块 (5 个)
- [x] [点赞操作](https://doc.west2.online/api-141535774.md): 仅需实现对视频的点赞
- [x] [点赞列表](https://doc.west2.online/api-141535775.md): 返回指定用户点赞的视频
- [x] [评论](https://doc.west2.online/api-141535776.md): 仅需实现对视频的评论，不需要对评论评论
- [x] [评论列表](https://doc.west2.online/api-141535777.md)
- [x] [删除评论](https://doc.west2.online/api-141551776.md): 不可删除他人评论

### 社交模块 (4 个)
- [x] [关注操作](https://doc.west2.online/api-141535778.md)
- [x] [关注列表](https://doc.west2.online/api-141535779.md): 根据 user_id 查看指定人的关注列表
- [x] [粉丝列表](https://doc.west2.online/api-141535780.md): 根据 user_id 查看指定人的粉丝列表
- [x] [好友列表](https://doc.west2.online/api-141535781.md): 查看当前登录用户的好友列表（互相关注）

---

## 不需要实现的 API (Bonus)

### 用户模块
- [ ] [获取 MFA qrcode](https://doc.west2.online/api-141546915.md): 获取绑定 MFA 时所需的二维码
- [ ] [绑定多因素身份认证(MFA)](https://doc.west2.online/api-141547523.md)
- [ ] [以图搜图](https://doc.west2.online/api-167998599.md): 用户上传图片原始数据

### 视频模块
- [ ] [视频流](https://doc.west2.online/api-141535771.md): 获取首页视频流

### 社交模块
- [ ] [聊天](https://doc.west2.online/websocket-3505546.md): WebSocket 聊天