package config

import (
	"github.com/2comjie/taoxi-server/app/Api/config/internal/router"
	"github.com/2comjie/taoxi-server/pkg/modules"
)

func Init(args modules.Modules) {
	router.Init(args)
}
