package items

import (
	"github.com/2comjie/taoxi-server/app/Api/items/internal/router"
	"github.com/2comjie/taoxi-server/app/Api/items/internal/store"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/2comjie/taoxi-server/pkg/modules"
)

func Init(args modules.Modules) {
	store.Init(external.MysqlGame())
	router.Init(args)
}
