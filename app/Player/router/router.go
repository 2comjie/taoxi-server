package router

import (
	"time"

	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/app/node"
	"github.com/2comjie/nova/logx"
	"github.com/2comjie/taoxi-server/app/Player/player"
)

type RouteArgs struct {
	PlayerActors *actor.Manager[*player.Player]
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

	initServiceRouter(args, root)
	initActorRouter(args, root)
}
