package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return "sess_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func newIdentifier() (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	return token[len("sess_"):], nil
}
func tokenDigest(t string) string {
	s := sha256.Sum256([]byte(t))
	return base64.RawURLEncoding.EncodeToString(s[:])
}
func validToken(t string) bool { return strings.HasPrefix(t, "sess_") && len(t) > 10 }
