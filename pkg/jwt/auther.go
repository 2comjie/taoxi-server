package jwt

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	PrivateKeyEnv     = "TAOXI_JWT_PRIVATE_KEY"
	PublicKeyEnv      = "TAOXI_JWT_PUBLIC_KEY"
	tokenIssuer       = "taoxi-api"
	gateAudience      = "taoxi-gate"
	gateTokenType     = "taoxi-login+jwt"
	gateTokenTTL      = 60 * time.Second
	accessAudience    = "taoxi-api"
	accessTokenType   = "taoxi-access+jwt"
	AccessTokenTTL    = 24 * time.Hour
	clockLeeway       = 3 * time.Second
	maxTokenSize      = 4 << 10
	maxClaimValueSize = 128
)

var (
	ErrInvalidToken            = errors.New("jwt: token无效或已过期")
	ErrPublicKeyNotInitialized = errors.New("jwt: 验签公钥未初始化")
	publicKeyValue             atomic.Value
)

type Claims struct {
	jwtlib.RegisteredClaims
}

type profile struct {
	audience string
	typ      string
	ttl      time.Duration
}

var (
	gateProfile   = profile{audience: gateAudience, typ: gateTokenType, ttl: gateTokenTTL}
	accessProfile = profile{audience: accessAudience, typ: accessTokenType, ttl: AccessTokenTTL}
)

func Generate(privateKey ed25519.PrivateKey, uid string) ([]byte, error) {
	return generate(privateKey, uid, gateProfile)
}

func Auth(token []byte) (string, error) {
	return auth(token, gateProfile)
}

func GenerateAccessToken(privateKey ed25519.PrivateKey, uid string) ([]byte, error) {
	return generate(privateKey, uid, accessProfile)
}

func AuthAccessToken(token []byte) (string, error) {
	return auth(token, accessProfile)
}

func InitPublicKey(publicKey ed25519.PublicKey) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New("jwt: Ed25519公钥无效")
	}
	publicKeyValue.Store(append(ed25519.PublicKey(nil), publicKey...))
	return nil
}

func auth(token []byte, tokenProfile profile) (string, error) {
	value := publicKeyValue.Load()
	if value == nil {
		return "", ErrPublicKeyNotInitialized
	}
	publicKey, ok := value.(ed25519.PublicKey)
	if !ok {
		return "", ErrPublicKeyNotInitialized
	}
	return verify(publicKey, token, tokenProfile)
}

func generate(privateKey ed25519.PrivateKey, uid string, tokenProfile profile) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("jwt: Ed25519私钥无效")
	}
	uid = strings.TrimSpace(uid)
	if uid == "" || len(uid) > maxClaimValueSize {
		return nil, errors.New("jwt: uid无效")
	}
	tokenID, err := newTokenID()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	claims := Claims{RegisteredClaims: jwtlib.RegisteredClaims{
		Issuer:    tokenIssuer,
		Audience:  jwtlib.ClaimStrings{tokenProfile.audience},
		Subject:   uid,
		ExpiresAt: jwtlib.NewNumericDate(now.Add(tokenProfile.ttl)),
		NotBefore: jwtlib.NewNumericDate(now),
		IssuedAt:  jwtlib.NewNumericDate(now),
		ID:        tokenID,
	}}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, claims)
	token.Header["typ"] = tokenProfile.typ
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("jwt: 签发token失败: %w", err)
	}
	return []byte(tokenString), nil
}

func verify(publicKey ed25519.PublicKey, tokenBytes []byte, tokenProfile profile) (string, error) {
	if len(publicKey) != ed25519.PublicKeySize || len(tokenBytes) == 0 || len(tokenBytes) > maxTokenSize {
		return "", ErrInvalidToken
	}
	claims := new(Claims)
	token, err := jwtlib.ParseWithClaims(
		string(tokenBytes),
		claims,
		func(token *jwtlib.Token) (any, error) {
			if token.Method != jwtlib.SigningMethodEdDSA {
				return nil, ErrInvalidToken
			}
			return publicKey, nil
		},
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}),
		jwtlib.WithIssuer(tokenIssuer),
		jwtlib.WithAudience(tokenProfile.audience),
		jwtlib.WithExpirationRequired(),
		jwtlib.WithLeeway(clockLeeway),
	)
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}
	tokenType, ok := token.Header["typ"].(string)
	if !ok || tokenType != tokenProfile.typ || !validClaims(claims, tokenProfile.ttl) {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

func validClaims(claims *Claims, maxTTL time.Duration) bool {
	if claims == nil || claims.Subject == "" || claims.ID == "" {
		return false
	}
	if claims.IssuedAt == nil || claims.NotBefore == nil || claims.ExpiresAt == nil {
		return false
	}
	if len(claims.Subject) > maxClaimValueSize || len(claims.ID) > maxClaimValueSize {
		return false
	}
	duration := claims.ExpiresAt.Sub(claims.IssuedAt.Time)
	return duration > 0 && duration <= maxTTL
}

func ParsePrivateKey(value string) (ed25519.PrivateKey, error) {
	decoded, err := decodeKey(value)
	if err != nil || len(decoded) != ed25519.PrivateKeySize {
		return nil, errors.New("jwt: Ed25519私钥必须是64字节Base64")
	}
	return append(ed25519.PrivateKey(nil), decoded...), nil
}

func ParsePublicKey(value string) (ed25519.PublicKey, error) {
	decoded, err := decodeKey(value)
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, errors.New("jwt: Ed25519公钥必须是32字节Base64")
	}
	return append(ed25519.PublicKey(nil), decoded...), nil
}

func decodeKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("jwt: key不能为空")
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}
	return nil, errors.New("jwt: key不是有效Base64")
}

func newTokenID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("jwt: 生成token id失败: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
