package modules

import (
	"github.com/2comjie/taoxi-server/pkg/asynqx"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type Modules struct {
	Engine      *gin.Engine
	ClientGroup *gin.RouterGroup
	OpenGroup   *gin.RouterGroup
	Cron        *cron.Cron
	AsynqServer *asynqx.Server

	ServerGroup *gin.RouterGroup
}
