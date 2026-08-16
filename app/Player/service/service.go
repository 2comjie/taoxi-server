package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/actor/actorDef"
	"github.com/2comjie/nova/actor/actorGuard"
	"github.com/2comjie/nova/deploy"
	"github.com/2comjie/taoxi-server/app/Player/player"
	"github.com/2comjie/taoxi-server/app/Player/router"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/2comjie/taoxi-server/pb/shared"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"google.golang.org/grpc"
)

func Init(app *deploy.NodeApp, rpcServer *grpc.Server) {
	actorSystem := actor.NewSystem(rpcServer)
	manager := actor.Register(actorSystem, actorDef.Type(pbShared.ActorType_Player), actorGuard.New(app.Instance().ID, external.RedisGame()), func(runCtx context.Context, pid actorDef.PID) (*player.Player, error) {
		return LoadPlayer(runCtx, cast.ToUint64(pid.Key))
	}, actor.RunnerConfig{
		QueueCap: 1000,
		UpdateDt: 2 * time.Second,
	})

	// 初始化路由
	router.Init(router.RouteArgs{PlayerActorManager: manager}, app.Router())

	if err := app.AddComponent(actorSystem); err != nil {
		panic(err)
	}
}

func LoadPlayer(ctx context.Context, uid uint64) (*player.Player, error) {
	bs, err := external.RedisGame().Get(ctx, fmt.Sprintf("player:{%d}", uid)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &player.Player{
				Level: 0,
			}, nil
		}
		return nil, err
	}

	pl := &player.Player{}
	_ = json.Unmarshal([]byte(bs), pl)
	return pl, nil
}
