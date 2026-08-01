package modules

import (
	"github.com/2comjie/wali/deploy"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type Modules struct {
	Engine      *gin.Engine
	ClientGroup *gin.RouterGroup
	OpenGroup   *gin.RouterGroup
	Cron        *cron.Cron
	App         *deploy.NodeApp
}
