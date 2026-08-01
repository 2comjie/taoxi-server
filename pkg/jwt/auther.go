package jwt

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	ticketIssuer   = "taoxi-api"
	ticketAudience = "taoxi-gate"
	ticketType     = "taoxi-login+jwt"
	ticketTTL      = 60 * time.Second
	clockLeeway    = 3 * time.Second
	maxTokenSize   = 4 << 10
)

var ErrInvalidToken = errors.New("jwt: token无效或已过期")

type Claims struct {
	jwtlib.RegisteredClaims
}

func Generate(privateKey ed25519.PrivateKey, uid string) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("jwt: Ed25519私钥无效")
	}
	if uid == "" {
		return nil, errors.New("jwt: uid不能为空")
	}

	tokenID, err := newTokenID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    ticketIssuer,
			Audience:  jwtlib.ClaimStrings{ticketAudience},
			Subject:   uid,
			ExpiresAt: jwtlib.NewNumericDate(now.Add(ticketTTL)),
			NotBefore: jwtlib.NewNumericDate(now),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, claims)
	token.Header["typ"] = ticketType
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("jwt: 签发token失败: %w", err)
	}
	return []byte(tokenString), nil
}

func Auth(publicKey ed25519.PublicKey, tokenBytes []byte) (string, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return "", ErrInvalidToken
	}
	if len(tokenBytes) == 0 || len(tokenBytes) > maxTokenSize {
		return "", ErrInvalidToken
	}

	claims := new(Claims)
	token, err := jwtlib.ParseWithClaims(
		string(tokenBytes),
		claims,
		func(token *jwtlib.Token) (any, error) {
			if token.Method.Alg() != jwtlib.SigningMethodEdDSA.Alg() {
				return nil, ErrInvalidToken
			}
			return publicKey, nil
		},
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}),
		jwtlib.WithIssuer(ticketIssuer),
		jwtlib.WithAudience(ticketAudience),
		jwtlib.WithExpirationRequired(),
		jwtlib.WithLeeway(clockLeeway),
	)
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}
	if tokenType, ok := token.Header["typ"].(string); !ok || tokenType != ticketType {
		return "", ErrInvalidToken
	}
	if !validClaims(claims) {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

func validClaims(claims *Claims) bool {
	if claims == nil || claims.Subject == "" || claims.ID == "" {
		return false
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return false
	}
	if len(claims.Subject) > 128 || len(claims.ID) > 128 {
		return false
	}
	return claims.ExpiresAt.Sub(claims.IssuedAt.Time) <= ticketTTL
}

func newTokenID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("jwt: 生成token id失败: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
