package token

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

var ErrMismatchedMethod = errors.New("mismatched signing method")
var ErrMismatchedClaims = errors.New("mismatched claims")

var ErrExpiredToken = errors.New("token is expired")

func validateRegisteredClaims(claims *jwt.RegisteredClaims) error {
	if claims.ExpiresAt == nil || time.Now().After(claims.ExpiresAt.Time) {
		return ErrExpiredToken
	}

	return nil
}
