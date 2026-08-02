package loginService

import (
	"context"

	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
)

type LoginProvider interface {
	Type() loginTypes.LoginType
	Authenticate(ctx context.Context, req *loginTypes.LoginReq) (*loginTypes.Identity, error)
}

type Manager struct {
}
