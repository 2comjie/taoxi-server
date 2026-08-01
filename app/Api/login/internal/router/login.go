package router

import (
	"github.com/2comjie/taoxi-server/app/Api/login/internal/service"
	logintypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/middleware/inout"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/taoxi-server/pkg/stderr"
)

func Init(args modules.Modules, manager *service.Manager) {
	args.ClientGroup.POST("login", inout.NoUidHandler(Login(manager)))
}

func Login(manager *service.Manager) func(*midef.Header, *logintypes.LoginReq) (*logintypes.LoginRsp, *stderr.Error) {
	return func(header *midef.Header, req *logintypes.LoginReq) (*logintypes.LoginRsp, *stderr.Error) {
		return manager.Login(header.Context(), req)
	}
}
