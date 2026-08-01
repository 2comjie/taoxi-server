package modules

import (
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type Modules struct {
	Engine      *gin.Engine
	ClientGroup *gin.RouterGroup
	OpenGroup   *gin.RouterGroup
	Cron        *cron.Cron
}
