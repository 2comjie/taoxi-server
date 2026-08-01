package nodeDeploy

import (
	"fmt"
	"sync"

	"github.com/2comjie/taoxi-server/flags"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/2comjie/taoxi-server/internal/deploy/instruction"
	netx "github.com/2comjie/wali/core/net"
	"github.com/2comjie/wali/deploy"
	redisLocator "github.com/2comjie/wali/locator/redis"
	redisRegistry "github.com/2comjie/wali/registry/redis"
)

var global *deploy.NodeApp
var once sync.Once

func Init(options ...deploy.Option) {
	once.Do(func() {
		var err error
		options = append(options, deploy.WithServiceName(flags.ServiceName))
		options = append(options, deploy.WithInstanceID(fmt.Sprintf("%s-%d", flags.ServiceName, flags.ServiceIndex)))

		// 1. 创建配置中心
		center, err := instruction.InitConfig()
		if err != nil {
			panic(err)
		}
		err = center.Load()
		if err != nil {
			panic(err)
		}
		options = append(options, deploy.WithConfig(center))

		// 2. 初始化日志/redis/mysql/mongo
		err = instruction.InitLogger(center)
		if err != nil {
			panic(err)
		}
		err = external.InitRedis(center)
		if err != nil {
			panic(err)
		}
		err = external.InitMysql(center)
		if err != nil {
			panic(err)
		}

		// 3. 初始化 注册中心/服务发现/locator
		options = append(options, deploy.WithRegistry(redisRegistry.NewRegistry(external.RedisRegistry())))
		options = append(options, deploy.WithDiscover(redisRegistry.NewDiscover(external.RedisRegistry())))
		options = append(options, deploy.WithLocator(redisLocator.NewProvider(external.RedisLocator())))

		// 4. 初始化 rpc 服务
		privateIP, err := netx.PrivateIP()
		if err != nil {
			panic(err)
		}
		options = append(options, deploy.WithRPCHost(privateIP))

		global, err = deploy.Node(options...)
		if err != nil {
			panic(err)
		}
	})
}

func App() *deploy.NodeApp {
	if global == nil {
		panic("nodeDeploy: App() called before Init()")
	}
	return global
}
