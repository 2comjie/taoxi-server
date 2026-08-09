package payment

import (
	paymentCron "github.com/2comjie/taoxi-server/app/Api/payment/internal/cron"
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
		panic(err)
	}

	paymentStore.Init(external.MysqlUser())
	paymentService.Init(external.RedisPayment(), args.AsynqServer)
	paymentCron.Init(args.Cron)
	router.Init(args)
}
