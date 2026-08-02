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
	args.ClientGroup.POST("account/bind", inout.UidHandler(Bind(manager)))
	args.ClientGroup.POST("account/unbind", inout.UidHandler(Unbind(manager)))
	args.ClientGroup.POST("account/delete", inout.UidHandler(DeleteAccount(manager)))
}

func Login(manager *loginService.Manager) inout.StdFunc[loginTypes.LoginReq, loginTypes.LoginRsp] {
	return func(header *midef.Header, req *loginTypes.LoginReq) (*loginTypes.LoginRsp, *stderr.Error) {
		return manager.Login(header.Context(), req)
	}
}

func Bind(manager *loginService.Manager) inout.StdFunc[loginTypes.LoginReq, loginTypes.BindRsp] {
	return func(header *midef.Header, req *loginTypes.LoginReq) (*loginTypes.BindRsp, *stderr.Error) {
		return manager.Bind(header.Context(), header.UID, req)
	}
}

func Unbind(manager *loginService.Manager) inout.StdFunc[loginTypes.LoginReq, loginTypes.UnbindRsp] {
	return func(header *midef.Header, req *loginTypes.LoginReq) (*loginTypes.UnbindRsp, *stderr.Error) {
		return manager.Unbind(header.Context(), header.UID, req)
	}
}

func DeleteAccount(manager *loginService.Manager) inout.StdFunc[loginTypes.DeleteAccountReq, loginTypes.DeleteAccountRsp] {
	return func(header *midef.Header, _ *loginTypes.DeleteAccountReq) (*loginTypes.DeleteAccountRsp, *stderr.Error) {
		return manager.DeleteAccount(header.Context(), header.UID)
	}
}
