package loginService

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	loginStore "github.com/2comjie/taoxi-server/app/Api/login/internal/store"
	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	"github.com/2comjie/taoxi-server/pkg/jwt"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx"
)

var ErrInvalidCredential = errors.New("login: 登录凭证无效")

type LoginProvider interface {
	Type() loginTypes.LoginType
	Authenticate(ctx context.Context, req *loginTypes.LoginReq) (*loginTypes.Identity, error)
}

type Manager struct {
	store      *loginStore.Store
	providers  map[loginTypes.LoginType]LoginProvider
	privateKey ed25519.PrivateKey
}

func NewManager(store *loginStore.Store, privateKey ed25519.PrivateKey) (*Manager, error) {
	if store == nil {
		return nil, errors.New("login: Store不能为空")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("login: JWT私钥无效")
	}
	return &Manager{
		store:      store,
		providers:  make(map[loginTypes.LoginType]LoginProvider),
		privateKey: append(ed25519.PrivateKey(nil), privateKey...),
	}, nil
}

func (m *Manager) Register(provider LoginProvider) error {
	if provider == nil {
		return errors.New("login: LoginProvider不能为空")
	}
	loginType := provider.Type()
	if _, exists := m.providers[loginType]; exists {
		return fmt.Errorf("login: LoginProvider重复注册 type=%d", loginType)
	}
	m.providers[loginType] = provider
	return nil
}

func (m *Manager) Login(ctx context.Context, req *loginTypes.LoginReq) (*loginTypes.LoginRsp, *stderr.Error) {
	if req == nil {
		return nil, stderr.New(http.StatusBadRequest, "登录请求不能为空")
	}

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
	uid, registered, err := m.store.FindOrCrateAccount(ctx, req.LoginType, *loginIdentity)
	if err != nil {
		if errors.Is(err, loginStore.ErrAccountDeleted) {
			return nil, stderr.New(http.StatusForbidden, "账号已注销")
		}
		logx.Errorf("login: 查找或者创建账号失败 type=%d err=%v", req.LoginType, err)
		return nil, stderr.New(http.StatusInternalServerError, "登录失败")
	}

	uidValue := strconv.FormatUint(uid, 10)
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
