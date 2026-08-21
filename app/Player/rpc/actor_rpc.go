package rpc

import (
	"context"

	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/actor/actorDef"
	"github.com/2comjie/taoxi-server/app/Player/player"
	pbPlayerActorRPC "github.com/2comjie/taoxi-server/pb/player/actor_rpc"
	"github.com/2comjie/taoxi-server/pkg/message_router"
)

func registerActorRPC(players *actor.Manager[*player.Player]) {
	message_router.RegActorRPC(players.RPC(), uint32(pbPlayerActorRPC.Route_GetState), getState)
}

func getState(current *player.Player, _ actorDef.PID, _ context.Context, req *pbPlayerActorRPC.GetStateReq, rsp *pbPlayerActorRPC.GetStateRsp) error {
	return current.GetState(req, rsp)
}
