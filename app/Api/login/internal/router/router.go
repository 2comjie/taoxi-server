package router

import (
	loginService "github.com/2comjie/taoxi-server/app/Api/login/internal/service"
	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/middleware/inout"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/taoxi-server/pkg/stderr"
)

func Init(args modules.Modules, manager *loginService.Manager) {
	args.ClientGroup.POST("login", inout.NoUidHandler(Login(manager)))
}

func Login(manager *loginService.Manager) inout.StdFunc[loginTypes.LoginReq, loginTypes.LoginRsp] {
	return func(header *midef.Header, req *loginTypes.LoginReq) (*loginTypes.LoginRsp, *stderr.Error) {
		return manager.Login(header.Context(), req)
	}
}
