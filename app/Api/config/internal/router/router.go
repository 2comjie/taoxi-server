package router

import (
	"github.com/2comjie/taoxi-server/app/Api/config/internal/service"
	configTypes "github.com/2comjie/taoxi-server/app/Api/config/types"
	gateDeploy "github.com/2comjie/taoxi-server/internal/deploy/gate"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/middleware/inout"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/taoxi-server/pkg/stderr"
)

func Init(args modules.Modules) {
	configGroup := args.ClientGroup.Group("config")
	configGroup.POST("gate_address", inout.UidHandler[configTypes.GetGateAddressReq, configTypes.GetGateAddressRsp](handleGateAddress))
}

func handleGateAddress(ctx *midef.Header, req *configTypes.GetGateAddressReq) (*configTypes.GetGateAddressRsp, *stderr.Error) {
	endpoint, err := service.RdGateAddress(ctx.Context(), ctx.Uid)
	if err != nil {
		return nil, err
	}
	return &configTypes.GetGateAddressRsp{
		WsAddress:  endpoint.MetaData[gateDeploy.MetaWSExternalAddress],
		KcpAddress: endpoint.MetaData[gateDeploy.MetaKCPExternalAddress],
		TcpAddress: endpoint.MetaData[gateDeploy.MetaTCPExternalAddress],
	}, nil
}
