# ai-speak\docs\zsc\用户账户与个人中心开发方案.m…

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

    userRepo := identity.NewMemoryRepository()
    authService := identity.NewService(userRepo)

    identity.NewHTTPHandler(authService).
        RegisterRoutes(mux)

    agent.NewHTTPHandler(agent.StubService{}).
        RegisterRoutes(mux)

    coaching.NewHTTPHandler().
        RegisterRoutes(mux)

    return mux
}
```

这里：

```go
userRepo := identity.NewMemoryRepository()
authService := identity.NewService(userRepo)
```

是在创建真实身份模块的依赖；

```go
identity.NewHTTPHandler(authService).RegisterRoutes(mux)
```

是在把身份模块接入总路由器。

更完整地说，路由装配通常包含：

- 创建 Repository；
- 创建 Service；
- 创建 Handler；
- 配置中间件；
- 注册 URL 路由；
- 将最终的 `http.Handler` 交给 `http.ListenAndServe`。

例如认证中间件可能这样装配：

```go
identityHandler := identity.NewHTTPHandler(authService)
identityHandler.RegisterRoutes(mux)

protectedPracticeHandler := identity.RequireAuth(authService)(
    coaching.NewHTTPHandler(),
)
```

不过以你当前项目为例，建议先把“业务依赖装配”和“路由注册”分开理解：

```text
依赖装配：
MemoryRepository -> AuthService -> IdentityHTTPHandler

路由注册：
IdentityHTTPHandler -> /v1/auth/register
IdentityHTTPHandler -> /v1/auth/login
IdentityHTTPHandler -> /v1/auth/logout
IdentityHTTPHandler -> /v1/me
```

另外，Flutter 中的“路由”是另一件事，指页面跳转，例如：

```dart
routes: {
  '/login': (_) => LoginPage(),
  '/register': (_) => RegisterPage(),
  '/practice': (_) => PracticePage(),
}
```

你这份方案里，“路由装配”主要指 Go 服务端的这一层，不是 Flutter 的页面路由。

> ```scss
> func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
>     mux.HandleFunc("POST /v1/auth/register", h.register)
>     mux.HandleFunc("POST /v1/auth/login", h.login)
>     mux.HandleFunc("POST /v1/auth/logout", h.logout)
>     mux.HandleFunc("GET /v1/me", h.currentUser)
> }
> ```
>
> 把这几个接口涉及到的inteface和方法定义输出一下

下面给出一套与开发方案一致的 Go 定义。这里统一采用：

- 注册成功只返回 `User`，不创建会话；
- 登录成功返回 `LoginResult`，其中包含一次性返回给客户端的原始 Token；
- 服务端内部只保存 Token 摘要；
- `logout` 和 `currentUser` 通过 Bearer Token 找到会话；
- Handler 不直接访问 Repository，只调用 `AuthService`。

注意代码块语言应使用 `go`，不是 `scss`。

## 1. 领域模型

```go
package identity

import "time"

// User 表示对外暴露的最小用户信息。
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// RegisterInput 表示注册请求参数。
type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginInput 表示登录请求参数。
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResult 表示登录成功后的响应结果。
type LoginResult struct {
	User      User      `json:"user"`
	Token     string    `json:"session_token"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthSession 表示服务端保存的会话记录。
// 这里不能保存原始 Token，只保存 Token 摘要。
type AuthSession struct {
	ID          string
	UserID      string
	TokenDigest []byte
	CreatedAt   time.Time
	ExpiresAt   time.Time
	RevokedAt   *time.Time
	Revocation  string
}

// UserRecord 表示用户及其认证凭据。
// 该结构只在身份模块内部使用，不直接作为 HTTP 响应。
type UserRecord struct {
	User         User
	PasswordHash string
}

// Actor 表示经过认证后写入请求上下文的身份信息。
type Actor struct {
	UserID    string
	SessionID string
}
```

## 2. Repository 接口

为了让内存实现和 PostgreSQL 实现使用同一套契约，建议不要把用户 Repository 和 Session Repository 拆得过于简单，而是直接定义身份模块需要的完整能力。

```go
package identity

import "context"

// Repository 定义身份模块的持久化能力。
// 阶段一由 MemoryRepository 实现，阶段二由 PostgreSQL Repository 实现。
type Repository interface {
	// CreateUserWithCredential 创建用户及其密码凭据。
	// 用户和凭据必须作为一个原子操作完成。
	CreateUserWithCredential(
		ctx context.Context,
		user User,
		passwordHash string,
	) (User, error)

	// FindUserByEmail 根据规范化邮箱查找用户及密码哈希。
	FindUserByEmail(
		ctx context.Context,
		canonicalEmail string,
	) (UserRecord, error)

	// FindUserByID 根据用户 ID 查找用户。
	FindUserByID(
		ctx context.Context,
		userID string,
	) (User, error)

	// CreateSession 保存一条新会话。
	// session.TokenDigest 必须已经由 Service 计算完成。
	CreateSession(
		ctx context.Context,
		session AuthSession,
	) error

	// FindSessionByTokenDigest 根据 Token 摘要查找会话。
	FindSessionByTokenDigest(
		ctx context.Context,
		tokenDigest []byte,
	) (AuthSession, error)

	// RevokeSession 撤销指定会话。
	// 重复撤销应当保持幂等。
	RevokeSession(
		ctx context.Context,
		sessionID string,
		reason string,
		now time.Time,
	) error
}
```

上面需要引入 `time`：

```go
import (
	"context"
	"time"
)
```

如果希望避免 Repository 接口暴露太多实现细节，也可以将 `now time.Time` 封装在 Service 中，让 Repository 自己使用数据库时间。但为了阶段一测试可控，直接传入时间通常更容易测试。

## 3. Service 接口

```go
package identity

import "context"

// AuthService 定义身份模块的业务用例。
type AuthService interface {
	// Register 校验并创建用户。
	// 注册成功后不创建会话。
	Register(
		ctx context.Context,
		input RegisterInput,
	) (User, error)

	// Login 校验邮箱和密码，并创建新的登录会话。
	Login(
		ctx context.Context,
		input LoginInput,
	) (LoginResult, error)

	// Authenticate 验证原始 Bearer Token，
	// 返回可信的用户和会话身份。
	Authenticate(
		ctx context.Context,
		rawToken string,
	) (Actor, error)

	// CurrentUser 根据已认证的用户 ID 返回当前用户。
	CurrentUser(
		ctx context.Context,
		userID string,
	) (User, error)

	// Logout 撤销指定会话。
	// 重复退出应当幂等。
	Logout(
		ctx context.Context,
		sessionID string,
	) error
}
```

这里 `Authenticate` 是必要的。因为 `/v1/me` 和 `/v1/auth/logout` 接收的是原始 Bearer Token，但 Service 内部应当使用 Token 摘要查询会话。

调用关系是：

```text
Authorization: Bearer sess_xxx
        |
        v
AuthService.Authenticate(ctx, "sess_xxx")
        |
        v
sha256("sess_xxx")
        |
        v
Repository.FindSessionByTokenDigest(...)
```

## 4. HTTPHandler 结构和路由注册

```go
package identity

import (
	"net/http"
)

// HTTPHandler 负责 HTTP 请求解析、响应编码和状态码映射。
type HTTPHandler struct {
	service AuthService
}

// NewHTTPHandler 创建身份模块 HTTP Handler。
func NewHTTPHandler(service AuthService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// RegisterRoutes 注册身份相关 HTTP 路由。
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/register", h.register)
	mux.HandleFunc("POST /v1/auth/login", h.login)
	mux.HandleFunc("POST /v1/auth/logout", h.logout)
	mux.HandleFunc("GET /v1/me", h.currentUser)
}
```

## 5. 四个 Handler 方法

### 注册

```go
func (h *HTTPHandler) register(
	writer http.ResponseWriter,
	request *http.Request,
) {
	var input RegisterInput

	if err := decodeJSON(request, &input); err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			ErrInvalidRequest,
		)
		return
	}

	user, err := h.service.Register(request.Context(), input)
	if err != nil {
		writeServiceError(writer, err)
		return
	}

	writeJSON(
		writer,
		http.StatusCreated,
		struct {
			User User `json:"user"`
		}{
			User: user,
		},
	)
}
```

注册接口的响应：

```json
{
  "user": {
    "id": "uuid",
    "email": "learner@example.com"
  }
}
```

### 登录

```go
func (h *HTTPHandler) login(
	writer http.ResponseWriter,
	request *http.Request,
) {
	var input LoginInput

	if err := decodeJSON(request, &input); err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			ErrInvalidRequest,
		)
		return
	}

	result, err := h.service.Login(request.Context(), input)
	if err != nil {
		writeServiceError(writer, err)
		return
	}

	writer.Header().Set("Cache-Control", "no-store")

	writeJSON(writer, http.StatusOK, result)
}
```

登录接口的成功响应：

```json
{
  "user": {
    "id": "uuid",
    "email": "learner@example.com"
  },
  "session_token": "sess_random_value",
  "token_type": "Bearer",
  "expires_at": "2026-10-03T12:00:00Z"
}
```

### 当前用户

```go
func (h *HTTPHandler) currentUser(
	writer http.ResponseWriter,
	request *http.Request,
) {
	rawToken, err := parseBearerToken(request)
	if err != nil {
		writeAuthenticationRequired(writer)
		return
	}

	actor, err := h.service.Authenticate(
		request.Context(),
		rawToken,
	)
	if err != nil {
		writeAuthenticationRequired(writer)
		return
	}

	user, err := h.service.CurrentUser(
		request.Context(),
		actor.UserID,
	)
	if err != nil {
		writeServiceError(writer, err)
		return
	}

	writer.Header().Set("Cache-Control", "no-store")

	writeJSON(
		writer,
		http.StatusOK,
		struct {
			User User `json:"user"`
		}{
			User: user,
		},
	)
}
```

### 退出登录

```go
func (h *HTTPHandler) logout(
	writer http.ResponseWriter,
	request *http.Request,
) {
	rawToken, err := parseBearerToken(request)
	if err != nil {
		writeAuthenticationRequired(writer)
		return
	}

	actor, err := h.service.Authenticate(
		request.Context(),
		rawToken,
	)
	if err != nil {
		writeAuthenticationRequired(writer)
		return
	}

	if err := h.service.Logout(
		request.Context(),
		actor.SessionID,
	); err != nil {
		writeServiceError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
```

## 6. Bearer Token 解析函数

建议把 Bearer 解析作为 HTTP 层的独立函数，不要散落在各个 Handler 中。

```go
package identity

import (
	"errors"
	"net/http"
	"strings"
)

var errInvalidAuthorization = errors.New("invalid authorization header")

func parseBearerToken(request *http.Request) (string, error) {
	header := strings.TrimSpace(
		request.Header.Get("Authorization"),
	)

	if header == "" {
		return "", errInvalidAuthorization
	}

	parts := strings.Fields(header)
	if len(parts) != 2 {
		return "", errInvalidAuthorization
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", errInvalidAuthorization
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errInvalidAuthorization
	}

	return token, nil
}
```

建议接受：

```http
Authorization: Bearer sess_abc
```

建议拒绝：

```http
Authorization: Basic sess_abc
Authorization: Bearer
Authorization: Bearer a b
Authorization: Bearer
```

## 7. 请求上下文版本的认证中间件

如果后续练习、简历、音频和报告接口也需要认证，建议增加中间件，让业务 Handler 不再重复解析 Bearer Token。

```go
package identity

import (
	"context"
	"net/http"
)

type contextKey string

const actorContextKey contextKey = "identity.actor"

// RequireAuth 验证 Bearer Token，并将 Actor 写入请求上下文。
func RequireAuth(service AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			rawToken, err := parseBearerToken(request)
			if err != nil {
				writeAuthenticationRequired(writer)
				return
			}

			actor, err := service.Authenticate(
				request.Context(),
				rawToken,
			)
			if err != nil {
				writeAuthenticationRequired(writer)
				return
			}

			ctx := context.WithValue(
				request.Context(),
				actorContextKey,
				actor,
			)

			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// ActorFromContext 获取经过认证的当前 Actor。
func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey).(Actor)
	return actor, ok
}
```

使用方式：

```go
protectedMeHandler := identity.RequireAuth(authService)(
	http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		actor, ok := identity.ActorFromContext(request.Context())
		if !ok {
			identity.WriteAuthenticationRequired(writer)
			return
		}

		// 使用 actor.UserID 查询当前用户资源。
	}),
)
```

不过第一阶段可以先让 `/v1/me` 和 `/v1/auth/logout` 在自身 Handler 中调用 `Authenticate`，等阶段四接入其他业务模块时再统一中间件。

## 8. 错误接口和错误映射

建议定义稳定领域错误：

```go
package identity

import "errors"

var (
	ErrInvalidRequest = errors.New("invalid request")

	ErrEmailAlreadyExists = errors.New(
		"email already exists",
	)

	ErrInvalidCredentials = errors.New(
		"invalid credentials",
	)

	ErrAuthenticationRequired = errors.New(
		"authentication required",
	)

	ErrUserNotFound = errors.New(
		"user not found",
	)

	ErrSessionNotFound = errors.New(
		"session not found",
	)
)
```

HTTP 层统一映射：

```go
func writeServiceError(
	writer http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		writeError(
			writer,
			http.StatusBadRequest,
			apiError{
				Code:      "invalid_request",
				Message:   "请求参数无效。",
				Retryable: false,
			},
		)

	case errors.Is(err, ErrEmailAlreadyExists):
		writeError(
			writer,
			http.StatusConflict,
			apiError{
				Code:      "account_registration_unavailable",
				Message:   "该邮箱无法注册。",
				Retryable: false,
			},
		)

	case errors.Is(err, ErrInvalidCredentials):
		writeError(
			writer,
			http.StatusUnauthorized,
			apiError{
				Code:      "invalid_credentials",
				Message:   "邮箱或密码不正确。",
				Retryable: false,
			},
		)

	case errors.Is(err, ErrAuthenticationRequired):
		writeAuthenticationRequired(writer)

	default:
		writeError(
			writer,
			http.StatusInternalServerError,
			apiError{
				Code:      "internal_error",
				Message:   "服务暂时不可用。",
				Retryable: true,
			},
		)
	}
}
```

错误结构：

```go
type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}
```

## 9. JSON 辅助方法

```go
package identity

import (
	"encoding/json"
	"net/http"
)

func decodeJSON(
	request *http.Request,
	target any,
) error {
	if request.Header.Get("Content-Type") != "application/json" {
		return ErrInvalidRequest
	}

	decoder := json.NewDecoder(
		http.MaxBytesReader(
			nil,
			request.Body,
			1<<20,
		),
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return ErrInvalidRequest
	}

	return nil
}

func writeJSON(
	writer http.ResponseWriter,
	status int,
	value any,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json",
	)
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(
	writer http.ResponseWriter,
	status int,
	err apiError,
) {
	writeJSON(
		writer,
		status,
		errorResponse{Error: err},
	)
}

func writeAuthenticationRequired(
	writer http.ResponseWriter,
) {
	writer.Header().Set(
		"WWW-Authenticate",
		`Bearer`,
	)

	writeError(
		writer,
		http.StatusUnauthorized,
		apiError{
			Code:      "authentication_required",
			Message:   "请先登录。",
			Retryable: false,
		},
	)
}
```

实际实现时，`http.MaxBytesReader` 的第一个参数应传入当前的 `http.ResponseWriter`，因此更合理的签名是：

```go
func decodeJSON(
	writer http.ResponseWriter,
	request *http.Request,
	target any,
) error {
	if request.Header.Get("Content-Type") != "application/json" {
		return ErrInvalidRequest
	}

	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		1<<20,
	)

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return ErrInvalidRequest
	}

	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return ErrInvalidRequest
	}

	return nil
}
```

对应 Handler 调用：

```go
if err := decodeJSON(writer, request, &input); err != nil {
	// ...
}
```

## 10. 推荐的最终依赖关系

```text
server/cmd/server/main.go
    |
    | 创建 MemoryRepository
    v
identity.Repository
    |
    | 注入
    v
identity.Service
    |
    | 注入
    v
identity.HTTPHandler
    |
    | RegisterRoutes
    v
http.ServeMux
    |
    +-- POST /v1/auth/register
    +-- POST /v1/auth/login
    +-- POST /v1/auth/logout
    +-- GET  /v1/me
```

启动装配可以写成：

```go
func buildRouter() http.Handler {
	mux := http.NewServeMux()

	repository := identity.NewMemoryRepository()
	authService := identity.NewService(repository)

	identity.NewHTTPHandler(authService).
		RegisterRoutes(mux)

	return mux
}
```

这里的关键点是：

```go
repository := identity.NewMemoryRepository()
authService := identity.NewService(repository)
identity.NewHTTPHandler(authService).RegisterRoutes(mux)
```

这三行分别完成：

1. 创建存储实现；
2. 创建业务服务并注入存储；
3. 创建 HTTP Handler 并注册四个接口。

这样后续切换 PostgreSQL 时，只需要替换第一行：

```go
repository := identity.NewPostgresRepository(db)
```

而 `Service`、`HTTPHandler` 和路由路径都不需要改变。
