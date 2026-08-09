package items

import (
	"github.com/2comjie/taoxi-server/app/Api/items/internal/router"
	"github.com/2comjie/taoxi-server/app/Api/items/internal/store"
	itemsConfig "github.com/2comjie/taoxi-server/internal/config/items"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	nodeDeploy "github.com/2comjie/taoxi-server/internal/deploy/node"
	"github.com/2comjie/taoxi-server/pkg/modules"
)

func Init(args modules.Modules) {
	if err := itemsConfig.Init(nodeDeploy.App().Config()); err != nil {
		panic(err)
	}
	store.Init(external.MysqlGame())
	router.Init(args)
}
