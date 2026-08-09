package apiBoost

import (
	"context"
	"errors"
	"github.com/2comjie/taoxi-server/app/Api/items"
	"net"
	"net/http"

	"github.com/2comjie/taoxi-server/app/Api/login"
	"github.com/2comjie/taoxi-server/app/Api/payment"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	nodeDeploy "github.com/2comjie/taoxi-server/internal/deploy/node"
	"github.com/2comjie/taoxi-server/pkg/asynqx"
	"github.com/2comjie/taoxi-server/pkg/middleware/auth"
	"github.com/2comjie/taoxi-server/pkg/middleware/extract"
	"github.com/2comjie/taoxi-server/pkg/middleware/inout"
	routeRecovery "github.com/2comjie/taoxi-server/pkg/middleware/recovery"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/wali/app"
	"github.com/2comjie/wali/core/help"
	"github.com/2comjie/wali/deploy"
	"github.com/2comjie/wali/logx"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func Init() {
	eg := gin.New()
	eg.Use(routeRecovery.Recovery())
	eg.Use(inout.LogxLogger())
	eg.Use(inout.AccessLog())
	eg.NoRoute(auth.Cors(), func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	systemCron := cron.New(cron.WithSeconds())

	clientGroup := eg.Group("api")
	clientGroup.Use(auth.Cors())
	clientGroup.Use(extract.ClientExtract())
	clientGroup.Use(auth.CheckClientSign())

	openGroup := eg.Group("open")
	openGroup.Use(auth.Cors())

	asynqServer := asynqx.NewServer(external.RedisAsynq, map[string]int{"default": 1})
	args := modules.Modules{
		Engine:      eg,
		ClientGroup: clientGroup,
		OpenGroup:   openGroup,
		Cron:        systemCron,
		AsynqServer: asynqServer,
	}

	// web 模块初始化
	webServer := &http.Server{
		Addr:    ":8080",
		Handler: eg,
	}
	webComponent := &app.CommonComponent{
		MName: "gin-server",
		MStart: func() error {
			listener, err := net.Listen("tcp", webServer.Addr)
			if err != nil {
				return err
			}
			help.SafeGo(func() {
				err := webServer.Serve(listener)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logx.Errorf("gin-server: 服务异常退出 err=%v", err)
				}
			})
			logx.Infof("gin-server: 服务器启动 %s", webServer.Addr)
			return nil
		},
		MShutdown: func(ctx context.Context) error {
			if err := webServer.Shutdown(ctx); err != nil {
				// 超时后强制断开连接
				return errors.Join(err, webServer.Close())
			}
			return nil
		},
	}
	cronComponent := &app.CommonComponent{
		MName: "system-cron",
		MStart: func() error {
			systemCron.Start()
			logx.Infof("system-cron: 定时任务启动")
			return nil
		},
		MShutdown: func(ctx context.Context) error {
			stopped := systemCron.Stop()
			select {
			case <-stopped.Done():
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	nodeDeploy.Init(deploy.WithComponents(webComponent, cronComponent, asynqServer))

	login.Init(args)
	items.Init(args)
	payment.Init(args)

	err := nodeDeploy.App().Run()

	if err != nil {
		panic(err)
	}
}
