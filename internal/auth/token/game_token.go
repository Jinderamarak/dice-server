package token

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
)

const SuperSecret = "my-256-bit-secret"

var (
	ErrGameMissingUserId = errors.New("missing user ID")
	ErrGameMissingGameId = errors.New("missing game ID")
)

type GameToken struct {
	UserId uuid.UUID `json:"userId"`
	GameId uuid.UUID `json:"gameId"`
	jwt.RegisteredClaims
}

func (token *GameToken) Validate() error {
	if token.UserId == uuid.Nil {
		return ErrGameMissingUserId
	}

	if token.GameId == uuid.Nil {
		return ErrGameMissingGameId
	}

	return validateRegisteredClaims(&token.RegisteredClaims)
}

func ValidateGameToken(tokenString string, secret []byte) (*GameToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &GameToken{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrMismatchedMethod
		}

		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*GameToken)
	if !ok {
		return nil, ErrMismatchedClaims
	}

	return claims, nil
}

func NewGameToken(userId, gameId uuid.UUID, issuer string, issuedAt, expiresAt time.Time) *GameToken {
	return &GameToken{
		UserId: userId,
		GameId: gameId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
}

func (token *GameToken) Sign(secret string) (string, error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, token)
	return jwtToken.SignedString(secret)
}
