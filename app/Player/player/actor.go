package player

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/2comjie/nova/actor/actorDef"
	"github.com/2comjie/nova/logx"
	"github.com/2comjie/nova/logx/logdef"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/spf13/cast"
)

type Player struct {
	logdef.ILogger `json:"-"`
	Level          int32 `json:"level"`

	LevelUpDt time.Duration `json:"level_up_dt"`
}

func (p *Player) OnStart(ctx actorDef.ActorStartCtx) error {
	p.ILogger = logx.WithField("uid", ctx.Self.Key)
	p.Infof("start")
	return nil
}

func (p *Player) OnUpdate(ctx actorDef.ActorUpdateCtx) time.Duration {
	p.Infof("update")
	p.LevelUpDt -= ctx.Delta
	if p.LevelUpDt <= 0 {
		p.LevelUpDt = 5 * time.Second
		p.Level++
		p.Infof("level up %d", p.Level)
	}
	return 0
}

func (p *Player) OnStop(ctx actorDef.ActorStopCtx) error {
	p.Infof("stop")
	jsonV, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return external.RedisGame().Set(ctx.Context, fmt.Sprintf("player:{%d}", cast.ToUint64(ctx.Self.Key)), jsonV, -1).Err()
}
