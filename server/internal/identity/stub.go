package identity

import (
	"context"
	"errors"
)

var errNotImplemented = errors.New("identity service is not implemented")

type StubAuthService struct{}

func (StubAuthService) Register(context.Context, RegisterInput) (User, error) {
	return User{}, errNotImplemented
}
func (StubAuthService) Login(context.Context, LoginInput) (LoginResult, error) {
	return LoginResult{}, errNotImplemented
}
func (StubAuthService) Authenticate(context.Context, string) (Actor, error) {
	return Actor{}, errNotImplemented
}
func (StubAuthService) CurrentUser(context.Context, string) (User, error) {
	return User{}, errNotImplemented
}
func (StubAuthService) Logout(context.Context, string) error { return errNotImplemented }
