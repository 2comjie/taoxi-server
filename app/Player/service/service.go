package service

import (
	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/deploy"
	"github.com/2comjie/taoxi-server/app/Player/player"
	"github.com/2comjie/taoxi-server/app/Player/router"
	playerRPC "github.com/2comjie/taoxi-server/app/Player/rpc"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type Service struct {
	app         *deploy.NodeApp
	redis       redis.UniversalClient
	actorSystem *actor.System
	players     *actor.Manager[*player.Player]
}

func Init(app *deploy.NodeApp, rpcServer *grpc.Server) {
	service := &Service{
		app:         app,
		redis:       external.RedisGame(),
		actorSystem: actor.NewSystem(rpcServer),
	}
	service.players = service.registerPlayerActor()

	playerRPC.Init(app, service.players, rpcServer)
	router.Init(router.RouteArgs{PlayerActors: service.players}, app.Router())

	if err := app.AddComponent(service.actorSystem); err != nil {
		panic(err)
	}
}
