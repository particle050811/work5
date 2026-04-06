# GitHub Actions 与代码质量门禁实践

> 日期：2026-04-06
> 主题：CodeQL、golangci-lint、单元测试、Docker 镜像构建与 CI 排错

---

## §1 问题背景

这次项目接入 Github Actions 之后，核心目标不再只是“代码能跑”，而是补齐一套最基本的工程门禁：

1. 安全扫描是否有自动化检查？
2. Go 代码规范和静态分析是否能在 PR 阶段拦截？
3. 已有单元测试是否会在 CI 中自动执行？
4. Docker 镜像是否能自动构建并推送？
5. 为什么本地 `golangci-lint` 能过，Github Actions 却失败？

这篇笔记围绕以上问题，整理当前项目的 CI 设计与踩坑记录。

---

## §2 当前项目有哪些 workflow

当前仓库已经拆成 4 条职责明确的 workflow：

| 工作流 | 作用 | 文件 |
|------|------|------|
| GolangCI-Lint | 静态检查与代码规范门禁 | `.github/workflows/golangci-lint.yml` |
| CodeQL | 安全漏洞扫描 | `.github/workflows/codeql.yml` |
| Unit Test | 自动执行 `go test ./...` | `.github/workflows/unit-test.yml` |
| Build Docker Image | 构建并推送 Docker 镜像 | `.github/workflows/docker-image.yml` |

关键代码位置：

- `.github/workflows/golangci-lint.yml:1`
- `.github/workflows/codeql.yml:1`
- `.github/workflows/unit-test.yml:1`
- `.github/workflows/docker-image.yml:1`

这样拆分的好处是：

- 单个 job 失败时更容易定位原因
- 安全、规范、测试、交付四类职责互不混淆
- 可以分别按需 rerun，而不是把所有逻辑塞进一个大 workflow

---

## §3 GolangCI-Lint workflow 做了什么

文件位置：

- `.github/workflows/golangci-lint.yml:1-42`

核心流程如下：

```text
push / pull_request / 手动触发
    ↓
checkout 仓库代码
    ↓
根据 shared/go.mod 安装 Go
    ↓
在 shared 目录执行 golangci-lint
    ↓
读取仓库根目录的 .golangci.yml
```

关键配置：

```yaml
on:
  push:
    branches: [main, master]
    paths:
      - "shared/**"
      - ".golangci.yml"
      - ".github/workflows/golangci-lint.yml"

jobs:
  lint:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: golangci/golangci-lint-action@v9
        with:
          version: v2.1
          working-directory: shared
          args: --config=../.golangci.yml
```

设计要点：

1. `paths` 过滤可以避免无关文件改动也触发 lint。
2. `working-directory: shared` 是因为 Go 模块根目录不在仓库根目录。
3. `args: --config=../.golangci.yml` 显式指定配置文件位置，避免 action 在子目录中找不到根目录配置。

---

## §4 `.golangci.yml` 开启了哪些检查

文件位置：

- `.golangci.yml:1-40`

当前配置：

```yaml
linters:
  disable-all: true
  enable:
    - errcheck
    - govet
    - ineffassign
    - misspell
    - revive
    - staticcheck
    - unconvert
```

这些检查的职责如下：

| Linter | 作用 |
|------|------|
| `errcheck` | 检查返回的 `error` 是否被处理 |
| `govet` | Go 官方静态检查，偏语言层面风险 |
| `ineffassign` | 无效赋值 |
| `misspell` | 英文拼写错误 |
| `revive` | 风格与可维护性规则 |
| `staticcheck` | 更强的静态分析 |
| `unconvert` | 检查多余类型转换 |

这里使用 `disable-all: true` 再显式 `enable`，目的是：

- 明确自己到底启用了哪些检查
- 避免默认规则集升级后行为漂移
- 便于答辩时解释“为什么开这些规则”

---

## §5 为什么关闭了 `unused-parameter`

这次最重要的一个经验是：

**不要为了过 lint，把所有没用到的 `ctx context.Context` 都机械改成 `_`。**

原因：

1. 在 Go 里，`Context` 作为第一个参数本身就是约定俗成的 API 形状。
2. 即使当前实现暂时没用到 `ctx`，保留 `ctx` 名称仍然有表达力，说明这个函数支持上下文传播。
3. 批量改成 `_` 虽然能消掉“未使用参数”告警，但会破坏接口可读性。

所以当前配置没有启用 `revive` 的 `unused-parameter` 规则，只保留更有价值的规则：

```yaml
linters-settings:
  revive:
    rules:
      - name: indent-error-flow
      - name: empty-block
      - name: superfluous-else
      - name: unreachable-code
      - name: var-declaration
      - name: atomic
      - name: bare-return
      - name: struct-tag
      - name: time-naming
```

文件位置：

- `.golangci.yml:29-40`

结论：

- 业务代码里的 `ctx` 应优先保留
- mock / fake / stub 这类明确忽略参数的场景，使用 `_` 是合理的

---

## §6 为什么 `swagger.go` 里写成 `_, _ = ctx.Write(...)`

文件位置：

- `shared/swagger/swagger.go:38-52`

当前代码：

```go
h.GET("/openapi/user.yaml", func(c context.Context, ctx *app.RequestContext) {
    ctx.Header("Content-Type", "application/x-yaml")
    _, _ = ctx.Write(userYAML)
})
```

原因不是业务逻辑变化，而是为了显式处理 `Write` 的返回值。

`ctx.Write(...)` 会返回：

```go
n, err := ctx.Write(userYAML)
```

如果直接裸调用：

```go
ctx.Write(userYAML)
```

在启用了 `errcheck` 的情况下会报错，因为它认为你“忘记处理 `error`”。

改成：

```go
_, _ = ctx.Write(userYAML)
```

表达的是：

- 我知道它会返回写入字节数和错误
- 这里显式选择忽略
- 这是有意识的忽略，不是漏处理

这类写法常见于：

- 输出很简单的只读响应
- 出错时通常也没有额外补救动作

---

## §7 为什么本地 lint 能过，CI 却失败

这次 CI 排错里最关键的坑是 `golangci-lint` 版本与 Go 版本不兼容。

项目 `go.mod` 声明的是：

```go
go 1.25.3
```

文件位置：

- `shared/go.mod:3`

而 Github Actions 里最初下载的 `golangci-lint v1.64.8` 是用 `go1.24` 构建的，导致报错：

```text
can't load config: the Go language version (go1.24) used to build golangci-lint
is lower than the targeted Go version (1.25.3)
```

这解释了为什么：

- 本地可以通过，因为本地的 `golangci-lint` 是用 `go1.25.3` 构建的
- CI 会失败，因为 action 下载的官方二进制较旧

最终修复方式是升级 workflow：

```yaml
- name: Run golangci-lint
  uses: golangci/golangci-lint-action@v9
  with:
    version: v2.1
    working-directory: shared
    args: --config=../.golangci.yml
```

文件位置：

- `.github/workflows/golangci-lint.yml:37-42`

经验总结：

1. 本地和 CI 的工具版本不一致时，优先看工具链版本。
2. 看到 `exit code 3` 不要只盯着表层状态，要展开原始日志。
3. workflow 里的 action 版本和 linter 版本都不应长期钉死在很旧的版本上。

---

## §8 为什么还要单独加 Unit Test workflow

文件位置：

- `.github/workflows/unit-test.yml:1-39`

当前流程：

```yaml
- name: Download dependencies
  run: go mod download

- name: Run unit tests
  run: go test ./...
```

为什么不能只靠 `golangci-lint`：

- `lint` 只能发现静态问题
- `go test` 才能验证行为是否正确

这两者关注点不同：

| 检查项 | 关注点 |
|------|------|
| `golangci-lint` | 代码质量、错误处理、风格、静态风险 |
| `go test ./...` | 逻辑行为是否符合预期 |

因此，只要仓库里已经有测试文件，就应该让 CI 自动执行。

---

## §9 为什么 Docker workflow 要调整 `latest` 推送顺序

文件位置：

- `.github/workflows/docker-image.yml:37-53`

当前写法：

```yaml
- name: Build and push commit tag
  id: build_image
  uses: docker/build-push-action@v6
  with:
    tags: particle050811/fanone-video:${{ github.sha }}

- name: Push latest tag after commit tag
  run: |
    docker buildx imagetools create \
      -t particle050811/fanone-video:latest \
      particle050811/fanone-video@${{ steps.build_image.outputs.digest }}
```

原因：

1. Docker Hub 不会对 `latest` 给予排序特权。
2. Docker Hub 按 `Newest` 排序时，看的只是推送时间。
3. 如果一次 build 同时推多个 tag，`latest` 不一定排在最上面。

所以这里改成：

- 先推提交哈希标签
- 再用同一个 digest 创建 `latest`

这样在 Docker Hub 上，`latest` 的推送时间更靠后，更稳定地显示在顶部。

---

## §10 当前这套 CI 的整体职责划分

可以把当前项目的 CI 理解成四层门禁：

```text
CodeQL
    负责安全扫描

golangci-lint
    负责静态分析和代码规范

Unit Test
    负责行为正确性验证

Docker Image
    负责交付产物构建
```

这四层互相补充，而不是互相替代。

如果只跑其中一条，都会留下明显空白：

- 没有 CodeQL：缺少安全扫描
- 没有 lint：低级错误和风格问题容易混进主分支
- 没有单测：逻辑回归没有自动兜底
- 没有 Docker 构建：交付链路没有验证

---

## §11 关键代码位置

- `.github/workflows/golangci-lint.yml:1`
- `.github/workflows/codeql.yml:1`
- `.github/workflows/unit-test.yml:1`
- `.github/workflows/docker-image.yml:1`
- `.golangci.yml:1`
- `shared/go.mod:3`
- `shared/swagger/swagger.go:38`

---

## §12 推荐阅读

- GolangCI-Lint 官方文档：https://golangci-lint.run/
- GolangCI-Lint Action：https://github.com/golangci/golangci-lint-action
- GitHub CodeQL 官方文档：https://docs.github.com/code-security/code-scanning/automatically-scanning-your-code-for-vulnerabilities-and-errors/about-code-scanning-with-codeql
- GitHub Actions 官方文档：https://docs.github.com/actions
- Go Code Review Comments（Context 相关约定）：https://go.dev/wiki/CodeReviewComments

---

## §13 总结

这次接入 CI 的核心收获有三点：

1. Github Actions 不应该只做“能不能构建”，还要覆盖安全、规范、测试和交付四个层面。
2. `golangci-lint` 配置不能只追求“全过”，还要兼顾 Go 代码本身的习惯用法，例如保留 `ctx` 的 API 形状。
3. CI 与本地不一致时，优先检查工具链版本、工作目录和配置文件路径，而不是盲目改业务代码。
