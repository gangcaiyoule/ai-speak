package identity

import (
	"context"
	"errors"
)

var errNotImplemented = errors.New("identity service is not implemented")

// StubAuthService 是架构阶段使用的身份服务空实现。
type StubAuthService struct{}

// Register 返回“未实现”占位错误。
func (StubAuthService) Register(context.Context, RegisterInput) (Session, error) {
	return Session{}, errNotImplemented
}

// Login 返回“未实现”占位错误。
func (StubAuthService) Login(context.Context, LoginInput) (Session, error) {
	return Session{}, errNotImplemented
}

// Logout 返回“未实现”占位错误。
func (StubAuthService) Logout(context.Context, string) error { return errNotImplemented }

// CurrentUser 返回“未实现”占位错误。
func (StubAuthService) CurrentUser(context.Context, string) (User, error) {
	return User{}, errNotImplemented
}
