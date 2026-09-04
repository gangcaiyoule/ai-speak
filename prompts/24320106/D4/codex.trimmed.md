# E:\Project\AI英语口语陪练\实践课作业\ai-speak\docs\zsc\用户账户与个人中心开发方案.m…

> E:\Project\AI英语口语陪练\实践课作业\ai-speak\docs\zsc\用户账户与个人中心开发方案.md
> 阅读这个开发方案，有什么建议

<details><summary>7 previous messages</summary>

> 我先通读这份开发方案，并结合仓库现有结构做一次偏工程落地的评审，重点看范围边界、数据模型、权限安全、接口契约、前端状态和验收标准是否有遗漏。 daarna我会按优先级给出可执行的修改建议。
>
> <details><summary>Ran a command</summary>
>
> - Ran `Get-Content -Raw -Encoding UTF8 'docs/zsc/用户账户与个人中心开发方案.md'`
>
> </details>
>
> 方案本身已经把阶段边界和安全底线写得比较清楚了。我正在对照当前仓库的 `identity`、路由装配、Flutter 目录和已有测试，确认哪些建议是“方案缺口”，哪些其实已经被现有实现约束，避免给出脱离代码现状的意见。
>
> <details><summary>Ran 6 commands</summary>
>
> - Ran `rg -n "StubAuthService|identity|/v1/auth|/v1/me|practice|net/http|gin" server mobile README* docs -g '!**/.dart_tool/**' -g '!**/build/**'`
> - Ran `rg --files | rg '(^|/|\\)(identity|mobile|server|README|go.mod|pubspec|AGENTS)'`
> - Ran `git status --short; git log -5 --oneline --decorate`
> - Ran `Get-Content -Raw -Encoding UTF8 server/internal/identity/model.go; Get-Content -Raw -Encoding UTF8 server/internal/identity/ports.go; Get-Content -Raw -Encoding UTF8 server/internal/identity/http.go; Get-Content -Raw -Encoding UTF8 server/internal/identity/stub.go`
> - Ran `Get-Content -Raw -Encoding UTF8 server/cmd/server/main.go; Get-Content -Raw -Encoding UTF8 server/go.mod; Get-Content -Raw -Encoding UTF8 mobile/lib/identity/identity_client.dart; Get-Content -Raw -Encoding UTF8 mobile/lib/main.dart; Get-Content -Raw -Encoding UTF8 mobile/lib/app/speak_up_app.dart`
> - Ran `Get-Content -Raw -Encoding UTF8 server/internal/identity/http_test.go; Get-Content -Raw -Encoding UTF8 server/cmd/server/main_test.go; Get-Content -Raw -Encoding UTF8 README.md`
>
> </details>
>
> 代码对照后有两个很明确的“先改方案再开工”的问题：当前身份接口仍把注册定义成“创建会话”，而文档又要求注册不登录；另外现有 Repository 接口没有凭据、Token 摘要和鉴权查询能力，按现状直接进入阶段一，阶段二很可能要重写接口。我会把这些和其他建议按优先级整理出来。
>
> 整体判断：这份方案的方向是对的，尤其是“先稳定 API 契约、身份由服务端会话解析、只保存 Token 摘要、分阶段替换存储”的思路比较稳健。它已经接近一份可执行的技术方案，但在开始第 1 个 Issue 前，建议先修正几处契约和架构问题，否则阶段一做完后，阶段二和 Flutter 接入很可能需要返工。
>
> **最高优先级：先统一注册和会话契约**
>
> 当前文档存在直接矛盾：
>
> - 第 3.1 节说注册成功只返回 `user`，且“不自动创建会话”。
> - 第 4.1 节的 `AuthService.Register` 仍返回 `Session`。
> - 当前代码的 `Register` 注释也写成了“注册用户并创建登录会话”，见 [ports.go](./server/internal/identity/ports.go)。
> - 当前 Flutter 的 `IdentityClient.register` 也返回 `Future<Session>`，见 [identity_client.dart](./mobile/lib/identity/identity_client.dart)。
>
> 建议明确采用以下契约：
>
> ```go
> Register(ctx context.Context, input RegisterInput) (User, error)
> Login(ctx context.Context, input LoginInput) (LoginResult, error)
> ```
>
> 其中：
>
> ```go
> type LoginResult struct {
>     User      User      `json:"user"`
>     Token     string    `json:"session_token"`
>     TokenType string    `json:"token_type"`
>     ExpiresAt time.Time `json:"expires_at"`
> }
> ```
>
> 持久化用的会话模型不要直接复用 API 返回模型，避免原始 Token 被意外保存或记录：
>
> ```go
> type AuthSession struct {
>     ID          string
>     UserID      string
>     TokenDigest []byte
>     CreatedAt   time.Time
>     ExpiresAt   time.Time
>     RevokedAt   *time.Time
> }
> ```
>
> **最高优先级：现在就设计好阶段二需要的 Repository 能力**
>
> 当前 [ports.go](./server/internal/identity/ports.go) 只有：
>
> - 创建用户
> - 按 ID 查用户
> - 创建会话
> - 删除会话
>
> 但实际需求还需要：
>
> - 按规范化邮箱查询用户
> - 保存和验证密码哈希
> - 按 Token 摘要查询会话
> - 判断会话是否过期或撤销
> - 撤销会话，而不是物理删除
> - 注册时用户和凭据在同一事务中创建
>
> 如果阶段一先围绕当前接口实现，阶段二会被迫修改 `AuthService` 和 Repository 契约。建议现在就改成更接近领域用例的接口，例如：
>
> ```go
> type IdentityRepository interface {
>     CreateUserWithCredential(ctx context.Context, user User, passwordHash string) (User, error)
>     FindUserByEmail(ctx context.Context, canonicalEmail string) (UserRecord, error)
>     FindSessionByTokenDigest(ctx context.Context, digest []byte) (AuthSession, error)
>     CreateSession(ctx context.Context, session AuthSession) error
>     RevokeSession(ctx context.Context, sessionID, reason string) error
> }
> ```
>
> 内存实现和 PostgreSQL 实现都遵循这一契约，阶段二就能真正做到“只替换 Repository”。
>
> **密码方案需要补充 bcrypt 的 72 字节限制**
>
> 文档规定密码最多 128 个字符，但 bcrypt 实际只处理前 72 字节。这样会出现两个不同密码被当成同一个密码的问题，尤其是中文或其他多字节字符。
>
> 需要在方案中明确二选一：
>
> 1. 将密码限制为最多 72 字节，并明确按 UTF-8 字节数检查；
> 2. 先对密码做标准化和 SHA-256，再将摘要交给 bcrypt；
> 3. 阶段一直接使用 Argon2id，避免 bcrypt 的输入长度限制。
>
> 如果继续使用 bcrypt，我更建议把规则写成“密码长度 8 至 72 个 UTF-8 字节”，并增加测试覆盖多字节字符。不要只写“8 至 128 个字符”。
>
> **把认证请求和会话上下文的流程写得更具体**
>
> 文档提到了把 `UserID` 和 `SessionID` 写入上下文，但目前接口设计还是：
>
> ```go
> CurrentUser(ctx, sessionID string)
> Logout(ctx, sessionID string)
> ```
>
> 建议明确中间件和 Handler 的职责：
>
> 1. 从 `Authorization` 解析 Bearer Token。
> 2. 对 Token 做 SHA-256。
> 3. Repository 根据摘要读取会话。
> 4. 检查 `revoked_at` 和 `expires_at`。
> 5. 将 `UserID`、`SessionID` 写入 context。
> 6. `/v1/me` 从 context 取 UserID。
> 7. `/logout` 从 context 取 SessionID 并撤销。
>
> 这样可以避免不同 Handler 自己解析 Token，减少认证逻辑分散。
>
> 另外，所有 `401` 响应建议增加：
>
> ```http
> WWW-Authenticate: Bearer
> ```
>
> 这属于标准 Bearer 认证行为，也便于客户端和代理识别。
>
> **错误契约还需要进一步收紧**
>
> 当前方案的错误结构是：
>
> ```json
> {
>   "error": {
>     "code": "...",
>     "message": "...",
>     "retryable": false
>   }
> }
> ```
>
> 但现有 Handler 返回的是顶层 `code` 和 `message`，见 [http.go](./server/internal/identity/http.go)。建议在方案里增加：
>
> - 每个错误码对应的 HTTP 状态码；
> - 字段错误如何表达，例如 `fields.email`、`fields.password`；
> - JSON 格式错误、未知字段、空请求体分别如何处理；
> - 服务端异常是否统一为 `internal_error`；
> - `retryable` 由服务端决定，客户端不得根据文字消息判断。
>
> 例如：
>
> ```json
> {
>   "error": {
>     "code": "invalid_request",
>     "message": "请求参数无效。",
>     "retryable": false,
>     "fields": {
>       "email": "邮箱格式不正确。"
>     }
>   }
> }
> ```
>
> 同时建议对注册、登录、`/v1/me` 和退出接口统一设置 `Cache-Control: no-store`，不只在登录成功响应中设置。
>
> **需要明确原始 Token 的生命周期**
>
> 方案要求“服务端不保存原始 Token”，但当前 `Session` 模型包含：
>
> ```go
> Token string `json:"token"`
> ```
>
> 建议明确：
>
> - 原始 Token 只在登录成功时生成一次；
> - 只通过响应返回一次；
> - 日志、错误、Tracing、Repository、数据库均不得出现；
> - 代码中将原始 Token 类型和持久化会话类型分开；
> - 可以考虑命名为 `RawToken`，提醒它不能进入持久化层。
>
> Token 生成建议同时定义编码方式。比如 32 字节随机值经过 base64url 编码，再加 `sess_` 前缀。验收测试要检查 Token 长度、字符集和前缀，而不是只检查“非空”。
>
> **内存 Repository 的并发语义要写清楚**
>
> “使用 `sync.RWMutex`”还不够。需要明确：
>
> - 同一规范化邮箱的注册必须在同一临界区内完成“检查 + 创建”；
> - 用户 ID 和会话 ID 如何生成；
> - `Create` 失败时是否保证内存状态不变；
> - `Logout` 是否幂等；
> - 过期会话是读取时判断，还是后台清理；
> - context 已取消时操作是否返回 `context.Canceled`。
>
> 并发验收建议加入：
>
> ```shell
> go test -race ./...
> ```
>
> 否则普通单元测试无法证明内存 Repository 没有数据竞争。
>
> **Flutter 部分目前缺少真正的应用装配方案**
>
> 当前应用启动仍然直接进入练习页：
>
> ```dart
> initialRoute: AppRoutes.practice
> ```
>
> 见 [speak_up_app.dart](./mobile/lib/app/speak_up_app.dart)。方案虽然提到了 `AuthGate`，但没有说明：
>
> - `SpeakUpApp` 如何同时接收 `IdentityClient` 和 `SceneClient`；
> - `AuthController` 的生命周期由谁管理；
> - 启动恢复期间显示什么；
> - 登录成功后如何构造带认证 Token 的练习客户端；
> - 练习接口收到 `401` 后如何通知 `AuthController`；
> - 注册页和登录页之间如何保留表单状态。
>
> 建议补一张依赖关系图或明确装配代码结构：
>
> ```text
> main
>  ├─ SessionStore
>  ├─ IdentityClient
>  ├─ AuthController
>  ├─ AuthGate
>  │   ├─ Login/Register
>  │   └─ AuthenticatedApp
>  │       └─ authenticated SceneClient
> ```
>
> 此外，当前环境变量叫 `SCENE_API_BASE_URL`，身份模块加入后建议统一为 `API_BASE_URL`，避免认证和练习各使用一套服务地址配置。
>
> **安全存储应提前确定技术依赖和平台限制**
>
> 文档写了 `SessionStore`，但没有指定实现。建议明确使用哪种安全存储，例如：
>
> - Android Keystore；
> - iOS Keychain；
> - Flutter 对应的 `flutter_secure_storage`。
>
> 同时写清楚以下行为：
>
> - 本地 Token 不放 `SharedPreferences`；
> - Token 读取异常时按未登录处理并记录可诊断但不敏感的错误；
> - 退出时服务端失败仍允许清除本地 Token；
> - Token 被服务端判定为 `401` 后必须清除；
> - 登录成功时先持久化 Token，持久化失败不能进入已登录页面。
>
> **验收矩阵还应增加几类关键边界测试**
>
> 当前测试矩阵已经比较完整，但建议补充：
>
> - 密码超过 bcrypt 输入限制；
> - 邮箱首尾空白、大小写、Unicode 和超长输入；
> - JSON 后还有额外内容；
> - `Content-Type` 缺失或错误；
> - 请求体超过限制；
> - `Authorization` 多个空格、错误 scheme、空 Token；
> - Token 摘要碰撞或长度异常；
> - 重复 Logout；
> - 服务端时间跳变；
> - 登录响应和错误响应不会写入日志；
> - `go test -race ./...`；
> - Flutter 真机/模拟器安全存储不可用时的失败路径。
>
> 端到端测试还要明确服务端是内存模式还是 PostgreSQL 模式，并提供固定启动命令，否则“注册、重启恢复”在阶段一中无法同时成立：内存模式重启后数据本来就会丢失。
>
> **阶段划分中有一个验收表述需要调整**
>
> 阶段一写了：
>
> > App 重启后可以恢复仍有效的会话。
>
> 如果用户账户和会话只存在内存 Repository，App 重启但 Go 服务不重启时可以恢复；如果服务也重启，则无法恢复。建议改成：
>
> - 客户端重启、服务端未重启时，可以恢复有效会话；
> - 服务端重启后数据丢失是阶段一已知限制；
> - PostgreSQL 阶段再验收服务端重启后的账户和会话持久化。
>
> **工作量估算偏乐观**
>
> 16 小时对 Go 领域、HTTP、Flutter 状态管理、安全存储、页面、联调和测试全部完成来说偏紧，尤其当前仓库还没有：
>
> - 真实身份服务；
> - Flutter `IdentityClient` HTTP 实现；
> - `SessionStore`；
> - `AuthController`；
> - `AuthGate`；
> - 登录和注册页面；
> - 带认证头的业务 Client；
> - 身份相关依赖和测试基础设施。
>
> 建议至少拆成两个可交付子任务：
>
> 1. Go 身份领域、内存存储、HTTP API 和测试；
> 2. Flutter 身份客户端、会话恢复、页面、AuthGate 和联调。
>
> 或者把估算改成“开发时间 + 环境准备/联调缓冲”，并把 PostgreSQL、Android/iOS 安全存储验证单独估算。
>
> 总结一下，最值得立即修改的是四点：
>
> 1. 统一“注册是否创建会话”的契约；
> 2. 现在就设计包含凭据和 Token 摘要查询的 Repository 接口；
> 3. 解决 bcrypt 的 72 字节限制；
> 4. 补全 Flutter 应用装配、`401` 传播和安全存储方案。
>
> 这四点明确后，方案就更适合直接拆成 Issue 开始实施了。
>
> 路由装配  这个是指什么
</details>

“路由装配”指的是：把各个业务模块的 HTTP Handler 创建出来，并把它们注册到服务器使用的路由器上。

在当前项目里，路由装配发生在 [server/cmd/server/main.go](./server/cmd/server/main.go) 的 `buildRouter()`：

```go
func buildRouter() http.Handler {
    mux := http.NewServeMux()

    mux.HandleFunc("GET /health", healthHandler)

    identity.NewHTTPHandler(identity.StubAuthService{}).
        RegisterRoutes(mux)

    agent.NewHTTPHandler(agent.StubService{}).
        RegisterRoutes(mux)

    coaching.NewHTTPHandler().
        RegisterRoutes(mux)

    return mux
}
```

这里做了三件事：

1. 创建一个总路由器 `mux`；
2. 创建身份、Agent、Coaching 等模块的 Handler；
3. 调用各模块的 `RegisterRoutes(mux)`，将具体 URL 注册到 `mux`。

例如身份模块内部可能注册：

```go
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("POST /v1/auth/register", h.register)
    mux.HandleFunc("POST /v1/auth/login", h.login)
    mux.HandleFunc("POST /v1/auth/logout", h.logout)
    mux.HandleFunc("GET /v1/me", h.currentUser)
}
```

这样，请求到达服务器时：

```text
POST /v1/auth/login
        |
        v
总路由器 mux
        |
        v
identity.HTTPHandler.login
        |
        v
AuthService.Login
        |
        v
Repository
```

所以“路由装配”不是指设计 URL，也不是指 Flutter 页面跳转，而是指在应用启动时完成服务端各模块的组合。

在你的方案中，我提到它，是因为第 4.1 节要求：

> 服务启动时把 `StubAuthService` 替换为真实 `Service + MemoryRepository`。

这就是一次路由装配调整。当前代码是：

```go
identity.NewHTTPHandler(identity.StubAuthService{}).
    RegisterRoutes(mux)
```

阶段一完成后，应该变成类似：

```go
func buildRouter() http.Handler {
    mux := http.NewServeMux()

    userRepo := identity.Ne