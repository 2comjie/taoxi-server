package router

import (
	"fmt"
	"time"

	"github.com/2comjie/taoxi-server/app/Player/player"
	pbPlayer "github.com/2comjie/taoxi-server/pb/player"
	"github.com/2comjie/taoxi-server/pkg/message_router"
	"github.com/2comjie/wali/actor"
	"github.com/2comjie/wali/actor/actorDef"
	"github.com/2comjie/wali/app/node"
	"github.com/2comjie/wali/logx"
)

type RouteArgs struct {
	PlayerActorSystem *actor.System[*player.Player]
}

func Init(args RouteArgs, root *node.Router) {
	_ = root.Use(func(next node.Handler) node.Handler {
		return func(context *node.Context) error {
			now := time.Now()
			logCtx := logx.WithField("client req", "").
				WithField("uid", context.Request.UID).
				WithField("route", context.Request.Route).
				WithField("gid", context.Request.GateInstanceID)
			logCtx.Info("收到客户端请求")
			err := next(context)
			logCtx.Infof("处理耗时 %v", time.Since(now))
			return err
		}
	})

	// 初始化每个模块的路由
	initPlayerRouter(args, root)
}

func initPlayerRouter(args RouteArgs, root *node.Router) {
	playerActorGroup := actor.NewRouteGroup[*player.Player](root, args.PlayerActorSystem, actor.ActivationLoad)
	message_router.RegActor(playerActorGroup, uint32(pbPlayer.ReqType_Hi), func(actorValue *player.Player, _ actorDef.PID, _ *node.Context, req *pbPlayer.HiReq, rsp *pbPlayer.HiRsp) error {
		logx.Infof("收到 hi 请求 %s", req.Name)
		rsp.Msg = fmt.Sprintf("hi %s %d", req.Name, actorValue.Level)
		return nil
	})
}
