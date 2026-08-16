package router

import (
	"fmt"
	"time"

	"github.com/2comjie/taoxi-server/app/Player/player"
	pbPlayer "github.com/2comjie/taoxi-server/pb/player"
	"github.com/2comjie/wali/actor"
	"github.com/2comjie/wali/actor/actorDef"
	"github.com/2comjie/wali/app/node"
	"github.com/2comjie/wali/logx"
	"google.golang.org/protobuf/proto"
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

	playerActorGroup.Handle(uint32(pbPlayer.ReqType_Hi), func(actorValue *player.Player, pid actorDef.PID, ctx *node.Context) error {
		hiReq := &pbPlayer.HiReq{}
		err := proto.Unmarshal(ctx.Request.Body, hiReq)
		if err != nil {
			return err
		}
		// 加个proto 的 消息转换？
		logx.Infof("收到 hi 请求 %s", hiReq.Name)
		hiRsp := &pbPlayer.HiRsp{
			Msg: fmt.Sprintf("hi %s %d", hiReq.Name, actorValue.Level),
		}
		data, err := proto.Marshal(hiRsp)
		if err != nil {
			return err
		}
		err = ctx.Reply(data)
		if err != nil {
			return err
		}
		return nil
	})

}
