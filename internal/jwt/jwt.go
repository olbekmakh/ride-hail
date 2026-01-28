package jwt

import (
	"errors"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Role string `json:"role"`
	jwtlib.RegisteredClaims
}

func ParseBearer(secret, authHeader string) (Claims, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return Claims{}, errors.New("missing bearer")
	}
	tok := strings.TrimPrefix(authHeader, "Bearer ")

	var c Claims
	_, err := jwtlib.ParseWithClaims(tok, &c, func(t *jwtlib.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return Claims{}, err
	}
	if c.Subject == "" || c.Role == "" {
		return Claims{}, errors.New("invalid claims")
	}
	return c, nil
}

func RequireRole(c Claims, role string) error {
	if c.Role != role {
		return errors.New("role forbidden")
	}
	return nil
}
