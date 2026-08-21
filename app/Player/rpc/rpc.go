package rpc

import (
	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/deploy"
	"github.com/2comjie/taoxi-server/app/Player/player"
	pbPlayerServiceRPC "github.com/2comjie/taoxi-server/pb/player/service_rpc"
	"google.golang.org/grpc"
)

type Service struct {
	pbPlayerServiceRPC.UnimplementedPlayerServiceServer
	app *deploy.NodeApp
}

func Init(app *deploy.NodeApp, players *actor.Manager[*player.Player], registrar grpc.ServiceRegistrar) {
	service := &Service{app: app}
	pbPlayerServiceRPC.RegisterPlayerServiceServer(registrar, service)
	registerActorRPC(players)
}
