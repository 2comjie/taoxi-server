package service

import (
	"context"

	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/actor/actorDef"
	"github.com/2comjie/nova/actor/actorGuard"
	"github.com/2comjie/taoxi-server/app/Player/player"
	"github.com/2comjie/taoxi-server/internal/diff_manager"
	playerRedisKey "github.com/2comjie/taoxi-server/internal/redis_key/player"
	pbPlayerActor "github.com/2comjie/taoxi-server/pb/player/actor"
	pbShared "github.com/2comjie/taoxi-server/pb/shared"
	"github.com/spf13/cast"
)

const playerDiffCount = 32

func (s *Service) registerPlayerActor() *actor.Manager[*player.Player] {
	return actor.Register(s.actorSystem, actorDef.Type(pbShared.ActorType_Player), actorGuard.New(s.app.Instance().ID, s.redis), s.loadPlayerActor, actor.RunnerConfig{
		QueueCap: 1000,
		UpdateDt: player.UpdateTick,
	})
}

func (s *Service) loadPlayerActor(ctx context.Context, pid actorDef.PID) (*player.Player, error) {
	uid := cast.ToUint64(pid.Key)
	storage := diff_manager.New[*pbPlayerActor.Player](s.redis, playerRedisKey.Diffs(uid), playerDiffCount, pbPlayerActor.ApplyPlayerDiff)
	value, _, err := storage.Load(ctx, &pbPlayerActor.Player{Uid: uid})
	if err != nil {
		return nil, err
	}
	realPlayer := player.New()
	realPlayer.Mount(pbPlayerActor.NewPlayerState(value))
	return realPlayer, nil
}
