package rpc

import (
	"context"

	"github.com/2comjie/nova/rpc/rpcerr"
	pbPlayerServiceRPC "github.com/2comjie/taoxi-server/pb/player/service_rpc"
)

func (s *Service) Ping(context.Context, *pbPlayerServiceRPC.PingReq) (*pbPlayerServiceRPC.PingRsp, rpcerr.Err) {
	return &pbPlayerServiceRPC.PingRsp{InstanceId: s.app.Instance().ID}, nil
}
