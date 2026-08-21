package player

import (
	"time"

	"github.com/2comjie/nova/actor/actorDef"
	"github.com/2comjie/nova/logx"
	"github.com/2comjie/nova/logx/logdef"
	"github.com/2comjie/taoxi-server/internal/util"
	pbPlayerActor "github.com/2comjie/taoxi-server/pb/player/actor"
)

const UpdateTick = 200 * time.Millisecond

type Player struct {
	logdef.ILogger

	state      *pbPlayerActor.PlayerState
	logicTime  time.Time
	needUnload bool
}

func New() *Player {
	return &Player{}
}

func (p *Player) Mount(state *pbPlayerActor.PlayerState) {
	p.state = state

}

func (p *Player) State() *pbPlayerActor.PlayerState {
	return p.state
}

func (p *Player) GetLogicTime() time.Time {
	return p.logicTime
}

func (p *Player) Recover() error {
	lastOfflineTime := p.state.GetLastOfflineTime()
	if lastOfflineTime == 0 {
		p.logicTime = time.Now()
	} else {
		p.logicTime = time.UnixMilli(lastOfflineTime)
	}
	return nil
}

func (p *Player) RequestUnload() {
	p.needUnload = true
}

func (p *Player) OnStart(ctx actorDef.ActorStartCtx) error {
	p.ILogger = logx.WithField("uid", ctx.Self.Key)
	if err := p.Recover(); err != nil {
		return err
	}

	if p.state.GetLastOfflineTime() != 0 {
		offlineDuration := time.Since(p.logicTime)
		start := time.Now()
		result := util.RunOffline(offlineDuration, p.update)
		p.Infof("离线计算完成 duration=%s cost=%s update_count=%d source_count=%v", offlineDuration, time.Since(start), result.UpdateCount, result.SourceCount)
	}
	p.Infof("start")
	return nil
}

func (p *Player) OnUpdate(ctx actorDef.ActorUpdateCtx) time.Duration {
	next := p.update(ctx.Delta)
	if p.needUnload {
		ctx.Unload()
	}
	return next.Dt
}

func (p *Player) OnStop(actorDef.ActorStopCtx) error {
	p.logicTime = time.Now()
	p.state.SetLastOfflineTime(p.logicTime.UnixMilli())
	p.Infof("stop")
	return nil
}

func (p *Player) update(delta time.Duration) util.NextUpdate {
	p.logicTime = p.logicTime.Add(delta)
	return util.MinNextUpdate(UpdateTick)
}

var _ actorDef.Actor = (*Player)(nil)
