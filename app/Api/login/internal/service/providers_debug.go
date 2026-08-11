package loginService

import (
	"context"

	loginStore "github.com/2comjie/taoxi-server/app/Api/login/internal/store"
	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	"github.com/spf13/cast"
)

type DebugLoginProvider struct {
	*BaseLoginProvider
}

func NewDebugLoginProvider(store *loginStore.Store) *DebugLoginProvider {
	return &DebugLoginProvider{
		BaseLoginProvider: NewBaseLoginProvider(store, loginTypes.LoginTypeDebug),
	}
}

func (p *DebugLoginProvider) Authenticate(ctx context.Context, req *loginTypes.LoginReq) (*loginTypes.Identity, error) {
	_, err := cast.ToUint64E(req.IdentityToken)
	if err != nil {
		return nil, err
	}
	return &loginTypes.Identity{
		AppID:  guestAppID,
		OpenID: req.IdentityToken,
	}, nil
}
func (p *DebugLoginProvider) Type() loginTypes.LoginType {
	return loginTypes.LoginTypeDebug
}
