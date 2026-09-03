package identity

import "context"

// UserRepository 定义身份模块保存和读取用户的能力。
type UserRepository interface {
	// Create 保存新用户并返回持久化结果。
	Create(ctx context.Context, user User) (User, error)
	// FindByID 按用户标识读取用户。
	FindByID(ctx context.Context, id string) (User, error)
	// CreateWithPasswordHash 保存用户及其密码哈希。
	CreateWithPasswordHash(ctx context.Context, user StoredUser) (User, error)
	// FindByEmailWithPasswordHash 按邮箱读取用户凭据。
	FindByEmailWithPasswordHash(ctx context.Context, email string) (StoredUser, error)
}

// SessionRepository 定义保存和删除登录会话的能力。
type SessionRepository interface {
	// Create 保存新会话并返回持久化结果。
	Create(ctx context.Context, session Session) (Session, error)
	// Delete 删除指定会话。
	Delete(ctx context.Context, id string) error
	// FindByTokenDigest 按令牌摘要读取有效会话。
	FindByTokenDigest(ctx context.Context, digest string) (Session, error)
}

// AuthService 定义注册、登录、退出和获取当前用户的用例。
type AuthService interface {
	// Register 注册用户并创建登录会话。
	Register(ctx context.Context, input RegisterInput) (Session, error)
	// Login 验证用户凭据并创建登录会话。
	Login(ctx context.Context, input LoginInput) (Session, error)
	// Logout 使指定登录会话失效。
	Logout(ctx context.Context, sessionID string) error
	// CurrentUser 返回指定会话关联的用户。
	CurrentUser(ctx context.Context, sessionID string) (User, error)
}
