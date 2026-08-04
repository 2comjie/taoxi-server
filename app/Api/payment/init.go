package payment

import (
	"fmt"

	"github.com/2comjie/taoxi-server/app/Api/payment/internal/router"
	paymentService "github.com/2comjie/taoxi-server/app/Api/payment/internal/service"
	paymentStore "github.com/2comjie/taoxi-server/app/Api/payment/internal/store"
	paymentConfig "github.com/2comjie/taoxi-server/internal/config/payment"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	nodeDeploy "github.com/2comjie/taoxi-server/internal/deploy/node"
	"github.com/2comjie/taoxi-server/pkg/modules"
)

func Init(args modules.Modules) {
	if err := paymentConfig.Init(nodeDeploy.App().Config()); err != nil {
		panic(fmt.Errorf("payment: 初始化商品配置失败: %w", err))
	}

	paymentStore.Init(external.MysqlUser())
	paymentService.Init(external.RedisUser())
	router.Init(args)
}
