package identity

import "strings"

func canonicalEmail(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if len(v) == 0 || len(v) > 254 || strings.Count(v, "@") != 1 || strings.ContainsAny(v, " \t\r\n") {
		return "", ErrInvalidRequest
	}
	p := strings.Split(v, "@")
	if p[0] == "" || p[1] == "" || !strings.Contains(p[1], ".") {
		return "", ErrInvalidRequest
	}
	return v, nil
}
