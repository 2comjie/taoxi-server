package loginService

import (
	"context"
	"fmt"

	loginStore "github.com/2comjie/taoxi-server/app/Api/login/internal/store"
	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	"github.com/google/uuid"
)

const guestAppID = "taoxi"

type GuestLoginProvider struct {
	*BaseLoginProvider
}

func NewGuestLoginProvider(store *loginStore.Store) *GuestLoginProvider {
	return &GuestLoginProvider{
		BaseLoginProvider: NewBaseLoginProvider(store, loginTypes.LoginTypeGuest),
	}
}

func (p *GuestLoginProvider) Authenticate(ctx context.Context, req *loginTypes.LoginReq) (*loginTypes.Identity, error) {
	guestID := req.GuestID
	if guestID == "" {
		for tryTimes := 0; tryTimes < 10; tryTimes++ {
			candidate := "custom_" + uuid.NewString()
			_, found, err := p.FindLoginRecord(ctx, guestAppID, candidate)
			if err != nil {
				return nil, err
			}
			if found {
				continue
			}
			guestID = candidate
			break
		}
		if guestID == "" {
			return nil, fmt.Errorf("login: 尝试生成游客ID失败")
		}
	}

	return &loginTypes.Identity{
		AppID:  guestAppID,
		OpenID: guestID,
	}, nil
}
