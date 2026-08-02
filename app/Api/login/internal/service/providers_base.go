package loginService

import (
	"context"

	loginStore "github.com/2comjie/taoxi-server/app/Api/login/internal/store"
	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
)

type BaseLoginProvider struct {
	store     *loginStore.Store
	loginType loginTypes.LoginType
}

func NewBaseLoginProvider(store *loginStore.Store, loginType loginTypes.LoginType) *BaseLoginProvider {
	return &BaseLoginProvider{
		store:     store,
		loginType: loginType,
	}
}

func (p *BaseLoginProvider) Type() loginTypes.LoginType {
	return p.loginType
}

func (p *BaseLoginProvider) FindLoginRecord(ctx context.Context, appID, openID string) (uid uint64, found bool, err error) {
	return p.store.FindLoginRecord(ctx, p.loginType, appID, openID)
}

func (p *BaseLoginProvider) FindOrCreateAccount(ctx context.Context, identity loginTypes.Identity) (uid uint64, registered bool, err error) {
	return p.store.FindOrCreateAccount(ctx, p.loginType, identity)
}

func (p *BaseLoginProvider) BindAccount(ctx context.Context, uid uint64, identity loginTypes.Identity) error {
	return p.store.BindIdentity(ctx, uid, p.loginType, identity)
}

func (p *BaseLoginProvider) UnbindAccount(ctx context.Context, uid uint64, identity loginTypes.Identity) error {
	return p.store.UnbindIdentity(ctx, uid, p.loginType, identity)
}
