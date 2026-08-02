package loginService

import (
	"context"
	"crypto/ed25519"
	"errors"
	"net/http"

	loginStore "github.com/2comjie/taoxi-server/app/Api/login/internal/store"
	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	"github.com/2comjie/taoxi-server/pkg/jwt"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx"
	"github.com/spf13/cast"
)

var ErrInvalidCredential = errors.New("login: 登录凭证无效")

type LoginProvider interface {
	Type() loginTypes.LoginType
	Authenticate(ctx context.Context, req *loginTypes.LoginReq) (*loginTypes.Identity, error)
	FindOrCreateAccount(ctx context.Context, identity loginTypes.Identity) (uid uint64, registered bool, err error)
	BindAccount(ctx context.Context, uid uint64, identity loginTypes.Identity) error
	UnbindAccount(ctx context.Context, uid uint64, identity loginTypes.Identity) error
}

type Manager struct {
	providers  map[loginTypes.LoginType]LoginProvider
	store      *loginStore.Store
	privateKey ed25519.PrivateKey
}

func NewManager(store *loginStore.Store, privateKey ed25519.PrivateKey) *Manager {
	if store == nil {
		panic("login: Store不能为空")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		panic("login: 私钥长度错误")
	}
	return &Manager{
		providers:  make(map[loginTypes.LoginType]LoginProvider),
		store:      store,
		privateKey: append(ed25519.PrivateKey(nil), privateKey...),
	}
}

func (m *Manager) Register(provider LoginProvider) {
	if provider == nil {
		panic("login: LoginProvider不能为空")
	}
	loginType := provider.Type()
	m.providers[loginType] = provider
}

func (m *Manager) Login(ctx context.Context, req *loginTypes.LoginReq) (*loginTypes.LoginRsp, *stderr.Error) {
	provider, exists := m.providers[req.LoginType]
	if !exists {
		return nil, stderr.New(http.StatusBadRequest, "不支持的登录方式")
	}
	loginIdentity, err := provider.Authenticate(ctx, req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredential) {
			return nil, stderr.New(http.StatusUnauthorized, "登录凭证无效")
		}
		logx.Errorf("login: 校验第三方登录凭证失败 type=%d err=%v", req.LoginType, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}
	uid, registered, err := provider.FindOrCreateAccount(ctx, *loginIdentity)
	if err != nil {
		if errors.Is(err, loginStore.ErrAccountDeleted) {
			return nil, stderr.New(http.StatusForbidden, "账号已注销")
		}
		logx.Errorf("login: 查找或者创建账号失败 type=%d err=%v", req.LoginType, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}

	uidValue := cast.ToString(uid)
	gateToken, err := jwt.Generate(m.privateKey, uidValue)
	if err != nil {
		logx.Errorf("login: 生成Gate token失败 uid=%d err=%v", uid, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}
	accessToken, err := jwt.GenerateAccessToken(m.privateKey, uidValue)
	if err != nil {
		logx.Errorf("login: 生成Access token失败 uid=%d err=%v", uid, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}

	logx.Infof("login: 登录成功 uid=%d type=%d register=%t", uid, req.LoginType, registered)
	return &loginTypes.LoginRsp{
		Uid:         uid,
		OpenId:      loginIdentity.OpenID,
		IsRegister:  registered,
		GateToken:   string(gateToken),
		AccessToken: string(accessToken),
	}, nil
}

func (m *Manager) Bind(ctx context.Context, uid uint64, req *loginTypes.LoginReq) (*loginTypes.BindRsp, *stderr.Error) {
	provider, exists := m.providers[req.LoginType]
	if !exists {
		return nil, stderr.New(http.StatusBadRequest, "不支持的登录方式")
	}
	loginIdentity, err := provider.Authenticate(ctx, req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredential) {
			return nil, stderr.New(http.StatusUnauthorized, "登录凭证无效")
		}
		return nil, stderr.New(http.StatusInternalServerError, "绑定失败")
	}

	err = provider.BindAccount(ctx, uid, *loginIdentity)
	if err != nil {
		switch {
		case errors.Is(err, loginStore.ErrAccountDeleted):
			return nil, stderr.New(http.StatusForbidden, "账号已注销")
		case errors.Is(err, loginStore.ErrIdentityBoundOtherAccount):
			return nil, stderr.New(http.StatusConflict, "该第三方账号已绑定其他玩家")
		case errors.Is(err, loginStore.ErrLoginTypeAlreadyBound):
			return nil, stderr.New(http.StatusConflict, "当前账号已绑定该登录方式")
		default:
			logx.Errorf("login: 绑定第三方账号失败 uid=%d type=%d err=%v", uid, req.LoginType, err)
			return nil, stderr.New(http.StatusInternalServerError, "绑定失败")
		}
	}

	logx.Infof("login: 绑定第三方账号成功 uid=%d type=%d", uid, req.LoginType)
	return &loginTypes.BindRsp{OpenId: loginIdentity.OpenID}, nil
}

func (m *Manager) Unbind(ctx context.Context, uid uint64, req *loginTypes.LoginReq) (*loginTypes.UnbindRsp, *stderr.Error) {
	provider, exists := m.providers[req.LoginType]
	if !exists {
		return nil, stderr.New(http.StatusBadRequest, "不支持的登录方式")
	}
	loginIdentity, err := provider.Authenticate(ctx, req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredential) {
			return nil, stderr.New(http.StatusUnauthorized, "登录凭证无效")
		}
		logx.Errorf("login: 校验取消绑定凭证失败 uid=%d type=%d err=%v", uid, req.LoginType, err)
		return nil, stderr.New(http.StatusInternalServerError, "取消绑定失败")
	}
	if err = provider.UnbindAccount(ctx, uid, *loginIdentity); err != nil {
		switch {
		case errors.Is(err, loginStore.ErrAccountDeleted):
			return nil, stderr.New(http.StatusForbidden, "账号已注销")
		case errors.Is(err, loginStore.ErrIdentityNotBound):
			return nil, stderr.New(http.StatusNotFound, "当前账号未绑定该第三方账号")
		case errors.Is(err, loginStore.ErrLastIdentity):
			return nil, stderr.New(http.StatusConflict, "至少保留一种登录方式")
		default:
			logx.Errorf("login: 取消绑定失败 uid=%d type=%d err=%v", uid, req.LoginType, err)
			return nil, stderr.New(http.StatusInternalServerError, "取消绑定失败")
		}
	}

	logx.Infof("login: 取消绑定成功 uid=%d type=%d", uid, req.LoginType)
	return &loginTypes.UnbindRsp{}, nil
}

func (m *Manager) DeleteAccount(ctx context.Context, uid uint64) (*loginTypes.DeleteAccountRsp, *stderr.Error) {
	if err := m.store.DeleteAccount(ctx, uid); err != nil {
		if errors.Is(err, loginStore.ErrAccountDeleted) {
			return nil, stderr.New(http.StatusForbidden, "账号已注销")
		}
		logx.Errorf("login: 注销账号失败 uid=%d err=%v", uid, err)
		return nil, stderr.New(http.StatusInternalServerError, "注销账号失败")
	}
	logx.Infof("login: 注销账号成功 uid=%d", uid)
	return &loginTypes.DeleteAccountRsp{}, nil
}
