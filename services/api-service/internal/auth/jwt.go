package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const Issuer = "asoc-auth"

type Claims struct {
	UserID int64  `json:"uid"`
	Email  string `json:"email,omitempty"`
	Name   string `json:"name,omitempty"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func ParseJWT(secret []byte, tokenStr string) (*Claims, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	if c.RegisteredClaims.Issuer != Issuer {
		return nil, errors.New("wrong issuer")
	}
	return c, nil
}

func Issue(secret []byte, userID int64, email, displayName, role string, ttl time.Duration) (string, error) {
	if len(secret) < 32 {
		return "", errors.New("jwt secret must be at least 32 bytes")
	}
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Name:   displayName,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}
