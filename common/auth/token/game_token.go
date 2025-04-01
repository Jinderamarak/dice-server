package token

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
)

var (
	ErrGameMissingUserID = errors.New("missing user ID")
	ErrGameMissingGameID = errors.New("missing game ID")
	ErrMissingServerID   = errors.New("missing server ID")
	ErrMissingServerHost = errors.New("missing server host")
)

type GameToken struct {
	UserID     uuid.UUID `json:"userId"`
	GameID     uuid.UUID `json:"gameId"`
	ServerID   uuid.UUID `json:"serverId"`
	ServerHost string    `json:"serverHost"`
	jwt.RegisteredClaims
}

func (token *GameToken) Validate() error {
	if token.UserID == uuid.Nil {
		return ErrGameMissingUserID
	}

	if token.GameID == uuid.Nil {
		return ErrGameMissingGameID
	}

	if token.ServerID == uuid.Nil {
		return ErrMissingServerID
	}

	if token.ServerHost == "" {
		return ErrMissingServerHost
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

func NewGameToken(userID, gameID, serverID uuid.UUID, serverHost string, issuer string, issuedAt, expiresAt time.Time) *GameToken {
	return &GameToken{
		UserID:     userID,
		GameID:     gameID,
		ServerID:   serverID,
		ServerHost: serverHost,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
}

func (token *GameToken) Sign(secret []byte) (string, error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, token)
	return jwtToken.SignedString(secret)
}
