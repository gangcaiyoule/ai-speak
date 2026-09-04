package identity

import "golang.org/x/crypto/bcrypt"

func validatePassword(v string) error {
	n := len([]byte(v))
	if n < 8 || n > 72 {
		return ErrInvalidRequest
	}
	return nil
}
func hashPassword(v string) (string, error) {
	h, e := bcrypt.GenerateFromPassword([]byte(v), bcrypt.DefaultCost)
	return string(h), e
}
func verifyPassword(h, v string) bool {
	return bcrypt.CompareHashAndPassword([]byte(h), []byte(v)) == nil
}
