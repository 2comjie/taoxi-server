package service

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/2comjie/taoxi-server/app/Api/login/internal/store"
	logintypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	taoxijwt "github.com/2comjie/taoxi-server/pkg/jwt"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx"
)

var ErrInvalidCredential = errors.New("login: 登录凭证无效")

type Provider interface {
	Type() logintypes.LoginType
	Authenticate(context.Context, *logintypes.LoginReq) (*logintypes.Identity, error)
}

type Manager struct {
	store      *store.Store
	providers  map[logintypes.LoginType]Provider
	privateKey ed25519.PrivateKey
}

func NewManager(accountStore *store.Store, privateKey ed25519.PrivateKey) (*Manager, error) {
	if accountStore == nil {
		return nil, errors.New("login: Store不能为空")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("login: JWT私钥无效")
	}
	return &Manager{
		store:      accountStore,
		providers:  make(map[logintypes.LoginType]Provider),
		privateKey: append(ed25519.PrivateKey(nil), privateKey...),
	}, nil
}

func (m *Manager) RegisterProvider(provider Provider) error {
	if provider == nil {
		return errors.New("login: Provider不能为空")
	}
	loginType := provider.Type()
	if _, exists := m.providers[loginType]; exists {
		return fmt.Errorf("login: 登录方式重复注册 type=%d", loginType)
	}
	m.providers[loginType] = provider
	return nil
}

func (m *Manager) Login(ctx context.Context, req *logintypes.LoginReq) (*logintypes.LoginRsp, *stderr.Error) {
	provider, exists := m.providers[req.LoginType]
	if !exists {
		return nil, stderr.New(http.StatusBadRequest, "不支持的登录方式")
	}
	identity, err := provider.Authenticate(ctx, req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredential) {
			return nil, stderr.New(http.StatusUnauthorized, "登录凭证无效")
		}
		logx.Errorf("login: 第三方登录校验失败 type=%d err=%v", req.LoginType, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}
	err = validateIdentity(identity)
	if err != nil {
		return nil, stderr.New(http.StatusUnauthorized, "登录凭证无效")
	}

	uid, registered, err := m.store.FindOrCreatePlayer(ctx, req.LoginType, *identity)
	if err != nil {
		if errors.Is(err, store.ErrPlayerDisabled) {
			return nil, stderr.New(http.StatusForbidden, "玩家账号不可用")
		}
		logx.Errorf("login: 查询或创建玩家失败 type=%d err=%v", req.LoginType, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}
	gateToken, err := taoxijwt.Generate(m.privateKey, uid)
	if err != nil {
		logx.Errorf("login: 生成Gate token失败 uid=%s err=%v", uid, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}
	accessToken, err := taoxijwt.GenerateAccessToken(m.privateKey, uid)
	if err != nil {
		logx.Errorf("login: 生成Access token失败 uid=%s err=%v", uid, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}
	rsp := &logintypes.LoginRsp{
		UID:         uid,
		OpenID:      identity.OpenID,
		IsRegister:  registered,
		GateToken:   string(gateToken),
		AccessToken: string(accessToken),
	}
	logx.Infof("login: 登录成功 uid=%s type=%d register=%t", uid, req.LoginType, registered)
	return rsp, nil
}

func validateIdentity(identity *logintypes.Identity) error {
	if identity == nil {
		return ErrInvalidCredential
	}
	identity.AppID = strings.TrimSpace(identity.AppID)
	identity.OpenID = strings.TrimSpace(identity.OpenID)
	identity.UnionID = strings.TrimSpace(identity.UnionID)
	if identity.AppID == "" || identity.OpenID == "" {
		return ErrInvalidCredential
	}
	if len(identity.AppID) > 255 || len(identity.OpenID) > 255 || len(identity.UnionID) > 255 {
		return ErrInvalidCredential
	}
	return nil
}
