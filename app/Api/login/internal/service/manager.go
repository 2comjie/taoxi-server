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
}

type Manager struct {
	providers  map[loginTypes.LoginType]LoginProvider
	privateKey ed25519.PrivateKey
}

func NewManager(privateKey ed25519.PrivateKey) *Manager {
	if len(privateKey) != ed25519.PrivateKeySize {
		panic("login: 私钥长度错误")
	}
	return &Manager{
		providers:  make(map[loginTypes.LoginType]LoginProvider),
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
