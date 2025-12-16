# 认证与双 Token 机制

> 日期：2025-12-16
> 主题：JWT 认证中间件、user_id 获取方式、安全设计

---

## §1 问题背景

在 `interaction_service.go` 的点赞接口中，获取当前用户使用的是：

```go
// 获取当前登录用户
userIDValue, exists := c.Get("user_id")
```

而不是从请求参数 `req.UserId` 获取。为什么？

---

## §2 两种获取用户 ID 的方式对比

### 2.1 `c.Get("user_id")` —— 当前登录用户（操作者）

| 属性 | 说明 |
|------|------|
| **来源** | JWT Token 经中间件验证后注入到请求上下文 |
| **含义** | **谁在执行这个操作** |
| **可信度** | 完全可信，无法伪造（除非 token 被盗） |

### 2.2 `req.UserId` —— 请求参数中的用户 ID

| 属性 | 说明 |
|------|------|
| **来源** | 客户端请求体/查询参数 |
| **含义** | **要查询/操作哪个用户的数据** |
| **可信度** | 不可信，客户端可以传任意值 |

### 2.3 使用场景对比

| 接口 | 使用方式 | 原因 |
|------|----------|------|
| `VideoLikeAction` (点赞) | `c.Get("user_id")` | 记录**谁**点的赞，必须是当前登录用户 |
| `PublishComment` (发评论) | `c.Get("user_id")` | 记录**谁**发的评论 |
| `DeleteComment` (删评论) | `c.Get("user_id")` | 验证是否是评论作者本人 |
| `ListLikedVideos` (点赞列表) | `req.UserId` | 可以查看**任意用户**的点赞列表 |
| `ListUserComments` (评论列表) | `req.UserId` | 可以查看**任意用户**的评论列表 |

### 2.4 安全角度

```go
// ❌ 危险：攻击者可以伪造任意用户点赞
userID := req.UserId  // 客户端可传入任何值

// ✅ 安全：只能以自己的身份点赞
userID := c.Get("user_id")  // 从已验证的 token 获取
```

**原则**：
- **写操作** 或 **权限校验** → 必须用 `c.Get("user_id")`
- **仅查询展示** 且允许查看他人数据 → 可用 `req.UserId`

---

## §3 JWT 中间件实现详解

### 3.1 中间件代码位置

`pkg/middleware/auth.go:15-62`

### 3.2 完整流程

```
请求进入
    ↓
① 从 Header 提取 Token（第 18 行）
   authHeader := c.GetHeader("Authorization")
    ↓
② 解析 Bearer 格式（第 28-37 行）
   "Bearer eyJhbGciOiJIUzI1NiIs..." → tokenString
    ↓
③ 验证并解析 JWT（第 40-41 行）
   claims, err := jwtMgr.ValidateAccessToken(tokenString)
    ↓
④ 存入上下文（第 57-58 行）
   c.Set("user_id", claims.UserID)    ← 写入
   c.Set("username", claims.Username)
    ↓
⑤ 继续执行后续 Handler（第 60 行）
   c.Next(ctx)
    ↓
⑥ Handler 中读取
   userIDValue, exists := c.Get("user_id")  ← 读取
```

### 3.3 中间件核心代码

```go
// pkg/middleware/auth.go

func AuthMiddleware() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        // ① 从 Authorization 头获取 Token
        authHeader := string(c.GetHeader("Authorization"))
        if authHeader == "" {
            c.JSON(consts.StatusUnauthorized, map[string]interface{}{
                "base": response.Unauthorized("请提供认证令牌"),
            })
            c.Abort()
            return
        }

        // ② 检查 Bearer 前缀
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(consts.StatusUnauthorized, map[string]interface{}{
                "base": response.Unauthorized("认证令牌格式错误"),
            })
            c.Abort()
            return
        }

        tokenString := parts[1]

        // ③ 验证 Token
        jwtMgr := auth.GetJWTManager()
        claims, err := jwtMgr.ValidateAccessToken(tokenString)
        if err != nil {
            // 处理过期、无效等错误...
            c.Abort()
            return
        }

        // ④ 将用户信息存入上下文
        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)

        // ⑤ 继续执行
        c.Next(ctx)
    }
}
```

---

## §4 JWT Claims 结构

### 4.1 自定义 Claims 定义

`pkg/auth/jwt.go:28-34`

```go
type Claims struct {
    UserID    uint      `json:"user_id"`    // 用户 ID
    Username  string    `json:"username"`   // 用户名
    TokenType TokenType `json:"token_type"` // access 或 refresh
    jwt.RegisteredClaims                    // 标准字段（exp, iat, nbf, iss）
}
```

### 4.2 Token 生成时写入 Claims

`pkg/auth/jwt.go:73-89`

```go
func (m *JWTManager) generateToken(userID uint, username string, tokenType TokenType, expiry time.Duration) (string, error) {
    now := time.Now()
    claims := &Claims{
        UserID:    userID,    // ← 登录时的用户 ID
        Username:  username,  // ← 登录时的用户名
        TokenType: tokenType,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
            IssuedAt:  jwt.NewNumericDate(now),
            NotBefore: jwt.NewNumericDate(now),
            Issuer:    "fanone-video-platform",
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secretKey)
}
```

---

## §5 双 Token 机制

### 5.1 两种 Token 类型

| 类型 | 有效期 | 用途 |
|------|--------|------|
| Access Token | 15 分钟 | 访问受保护资源 |
| Refresh Token | 7 天 | 刷新获取新的 Token 对 |

### 5.2 配置位置

`pkg/auth/jwt.go:50-54`

```go
return &JWTManager{
    secretKey:          []byte(secret),
    accessTokenExpiry:  15 * time.Minute,    // 访问令牌 15 分钟
    refreshTokenExpiry: 7 * 24 * time.Hour,  // 刷新令牌 7 天
}
```

### 5.3 Token 刷新流程

```go
// pkg/auth/jwt.go:146-153
func (m *JWTManager) RefreshTokens(refreshTokenString string) (newAccessToken, newRefreshToken string, err error) {
    // 验证 refresh token
    claims, err := m.ValidateRefreshToken(refreshTokenString)
    if err != nil {
        return "", "", err
    }
    // 生成新的 token 对
    return m.GenerateTokenPair(claims.UserID, claims.Username)
}
```

---

## §6 Hertz 上下文存取机制

### 6.1 `c.Set()` 和 `c.Get()`

Hertz 的 `app.RequestContext` 提供了键值存储功能：

```go
// 写入
c.Set("user_id", claims.UserID)  // key: string, value: any

// 读取
value, exists := c.Get("user_id")
if exists {
    userID := value.(uint)  // 需要类型断言
}
```

### 6.2 与 Go 标准 Context 的区别

| 特性 | `context.Context` | `app.RequestContext` |
|------|-------------------|---------------------|
| 用途 | 取消、超时、传值 | HTTP 请求处理 |
| 传值方式 | `WithValue()` 创建新 ctx | `Set()`/`Get()` 直接操作 |
| 生命周期 | 跨服务传递 | 仅限当前请求 |
| 在 Hertz 中 | 参数 `ctx` | 参数 `c` |

---

## §7 关键代码位置

| 功能 | 文件路径 | 行号 |
|------|----------|------|
| JWT Claims 定义 | `pkg/auth/jwt.go` | 28-34 |
| Token 生成 | `pkg/auth/jwt.go` | 57-70 |
| Token 验证 | `pkg/auth/jwt.go` | 117-129 |
| 认证中间件 | `pkg/middleware/auth.go` | 15-62 |
| 上下文写入 user_id | `pkg/middleware/auth.go` | 57 |
| Handler 读取 user_id | `biz/handler/v1/interaction_service.go` | 36-43 |

---

## §8 最佳实践

1. **写操作必须从 Token 获取身份**
   - 点赞、评论、关注等操作
   - 删除、修改等需要权限的操作

2. **读操作可以接受参数中的用户 ID**
   - 查看他人主页
   - 查看他人点赞/评论列表

3. **权限校验模式**
   ```go
   // 获取操作者身份
   operatorID := c.Get("user_id").(uint)

   // 获取资源所有者
   resource, _ := db.GetResourceByID(resourceID)

   // 校验权限
   if resource.OwnerID != operatorID {
       c.JSON(403, response.Forbidden("无权操作"))
       return
   }
   ```

4. **中间件链顺序**
   - 先执行 `AuthMiddleware` 注入用户信息
   - 再执行业务 Handler 读取用户信息

---

## §9 推荐阅读

- [JWT 官方介绍](https://jwt.io/introduction)
- [golang-jwt/jwt 库文档](https://github.com/golang-jwt/jwt)
- [Hertz 中间件文档](https://www.cloudwego.io/zh/docs/hertz/tutorials/basic-feature/middleware/)
- [OWASP 认证最佳实践](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
