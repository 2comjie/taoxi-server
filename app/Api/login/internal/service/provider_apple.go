package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	logintypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	appleJWKSURL      = "https://appleid.apple.com/auth/keys"
	appleIssuer       = "https://appleid.apple.com"
	appleJWKSCacheTTL = 10 * time.Minute
)

type AppleConfig struct {
	Audiences []string
}

type AppleProvider struct {
	audiences []string
	client    *http.Client

	keysMu        sync.Mutex
	keys          map[string]*rsa.PublicKey
	keysFetchedAt time.Time
}

type appleClaims struct {
	jwtlib.RegisteredClaims
}

type appleJWKSet struct {
	Keys []appleJWK `json:"keys"`
}

type appleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewAppleProvider(config AppleConfig) (*AppleProvider, error) {
	audiences := make([]string, 0, len(config.Audiences))
	for _, audience := range config.Audiences {
		if audience = strings.TrimSpace(audience); audience != "" {
			audiences = append(audiences, audience)
		}
	}
	if len(audiences) == 0 {
		return nil, errors.New("login: Apple audiences不能为空")
	}
	return &AppleProvider{
		audiences: audiences,
		client:    &http.Client{Timeout: 10 * time.Second},
		keys:      make(map[string]*rsa.PublicKey),
	}, nil
}

func (*AppleProvider) Type() logintypes.LoginType {
	return logintypes.LoginTypeApple
}

func (p *AppleProvider) Authenticate(ctx context.Context, req *logintypes.LoginReq) (*logintypes.Identity, error) {
	identityToken := strings.TrimSpace(req.IdentityToken)
	if identityToken == "" {
		return nil, ErrInvalidCredential
	}
	claims := new(appleClaims)
	token, err := jwtlib.ParseWithClaims(
		identityToken,
		claims,
		func(token *jwtlib.Token) (any, error) {
			if token.Method != jwtlib.SigningMethodRS256 {
				return nil, ErrInvalidCredential
			}
			kid, _ := token.Header["kid"].(string)
			if kid == "" {
				return nil, ErrInvalidCredential
			}
			return p.getKey(ctx, kid)
		},
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodRS256.Alg()}),
		jwtlib.WithIssuer(appleIssuer),
		jwtlib.WithExpirationRequired(),
		jwtlib.WithLeeway(3*time.Second),
	)
	if err != nil || !token.Valid || strings.TrimSpace(claims.Subject) == "" {
		return nil, ErrInvalidCredential
	}
	appID := p.matchAudience(claims.Audience)
	if appID == "" {
		return nil, ErrInvalidCredential
	}
	return &logintypes.Identity{AppID: appID, OpenID: claims.Subject}, nil
}

func (p *AppleProvider) matchAudience(tokenAudiences jwtlib.ClaimStrings) string {
	for _, audience := range tokenAudiences {
		if slices.Contains(p.audiences, audience) {
			return audience
		}
	}
	return ""
}

func (p *AppleProvider) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	p.keysMu.Lock()
	defer p.keysMu.Unlock()
	if key := p.keys[kid]; key != nil && time.Since(p.keysFetchedAt) < appleJWKSCacheTTL {
		return key, nil
	}
	if err := p.fetchKeys(ctx); err != nil {
		return nil, err
	}
	key := p.keys[kid]
	if key == nil {
		return nil, ErrInvalidCredential
	}
	return key, nil
}

func (p *AppleProvider) fetchKeys(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, appleJWKSURL, nil)
	if err != nil {
		return fmt.Errorf("login: 创建Apple公钥请求失败: %w", err)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("login: 获取Apple公钥失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("login: Apple公钥接口返回状态%d", response.StatusCode)
	}
	var set appleJWKSet
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&set); err != nil {
		return fmt.Errorf("login: 解析Apple公钥失败: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, item := range set.Keys {
		if item.Kty != "RSA" || item.Kid == "" || (item.Use != "" && item.Use != "sig") || (item.Alg != "" && item.Alg != jwtlib.SigningMethodRS256.Alg()) {
			continue
		}
		key, err := parseAppleRSAKey(item)
		if err != nil {
			return fmt.Errorf("login: 解析Apple公钥kid=%s失败: %w", item.Kid, err)
		}
		keys[item.Kid] = key
	}
	if len(keys) == 0 {
		return errors.New("login: Apple公钥列表为空")
	}
	p.keys = keys
	p.keysFetchedAt = time.Now()
	return nil
}

func parseAppleRSAKey(item appleJWK) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(item.N)
	if err != nil {
		return nil, err
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(item.E)
	if err != nil {
		return nil, err
	}
	modulus := new(big.Int).SetBytes(modulusBytes)
	exponent := new(big.Int).SetBytes(exponentBytes)
	if modulus.BitLen() < 2048 || !exponent.IsInt64() || exponent.Int64() <= 1 || exponent.Bit(0) == 0 || exponent.Int64() > int64(^uint(0)>>1) {
		return nil, errors.New("无效的RSA公钥")
	}
	return &rsa.PublicKey{N: modulus, E: int(exponent.Int64())}, nil
}
