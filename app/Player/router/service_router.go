package router

import (
	"fmt"

	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/actor/actorDef"
	"github.com/2comjie/nova/app/node"
	"github.com/2comjie/taoxi-server/app/Player/player"
	pbPlayer "github.com/2comjie/taoxi-server/pb/player"
	"github.com/2comjie/taoxi-server/pkg/message_router"
)

func initServiceRouter(args RouteArgs, root *node.Router) {
	message_router.Reg(root, uint32(pbPlayer.ReqType_Hi), hi)
	group := actor.NewRouteGroup[*player.Player](root, args.PlayerActors, actor.ActivationLoad)
	message_router.RegActor(group, uint32(pbPlayer.ReqType_LoadPlayer), loadPlayer)
}

func hi(_ *node.Context, req *pbPlayer.HiReq, rsp *pbPlayer.HiRsp) error {
	rsp.Msg = fmt.Sprintf("hi %s", req.Name)
	return nil
}

func loadPlayer(_ *player.Player, _ actorDef.PID, _ *node.Context, _ *pbPlayer.LoadPlayerReq, _ *pbPlayer.LoadPlayerRsp) error {
	return nil
}
