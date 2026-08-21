package router

import (
	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/actor/actorDef"
	"github.com/2comjie/nova/app/node"
	"github.com/2comjie/taoxi-server/app/Player/player"
	pbPlayer "github.com/2comjie/taoxi-server/pb/player"
	"github.com/2comjie/taoxi-server/pkg/message_router"
)

func initActorRouter(args RouteArgs, root *node.Router) {
	group := actor.NewRouteGroup[*player.Player](root, args.PlayerActors, actor.ActivationLoad)
	message_router.RegActor(group, uint32(pbPlayer.ReqType_Offload), offload)
}

func offload(current *player.Player, _ actorDef.PID, _ *node.Context, _ *pbPlayer.OffloadReq, _ *pbPlayer.OffloadRsp) error {
	current.RequestUnload()
	return nil
}
