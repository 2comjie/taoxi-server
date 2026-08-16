package router

import (
	"github.com/2comjie/taoxi-server/app/Api/mock/internal/service"
	mockTypes "github.com/2comjie/taoxi-server/app/Api/mock/types"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/middleware/inout"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/deploy"
)

func Init(args modules.Modules, app *deploy.NodeApp) {
	mockGroup := args.ServerGroup.Group("mock")
	mockGroup.POST("player", inout.NoUidHandler[mockTypes.GateMockReq, mockTypes.GateMockRsp](func(header *midef.Header, req *mockTypes.GateMockReq) (*mockTypes.GateMockRsp, *stderr.Error) {
		return service.GateMock(header.Context(), app, req)
	}))
}
