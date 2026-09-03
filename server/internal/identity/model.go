// Package identity 定义用户和会话相关的领域契约。
package identity

// User 表示最小用户身份信息。
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Session 表示不透明的用户登录会话。
type Session struct {
	ID    string `json:"id"`
	User  User   `json:"user"`
	Token string `json:"token"`
}

// RegisterInput 包含注册用户所需的数据。
type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginInput 包含用户登录所需的数据。
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// StoredUser 是服务端内部保存的用户凭据，不向 HTTP 响应暴露密码哈希。
type StoredUser struct {
	User         User
	PasswordHash string
}
