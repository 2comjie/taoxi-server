package gateDeploy

import (
	"fmt"
	"sync"

	"github.com/2comjie/taoxi-server/flags"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/2comjie/taoxi-server/internal/deploy/instruction"
	"github.com/2comjie/taoxi-server/pkg/jwt"
	"github.com/2comjie/taoxi-server/pkg/pprof"
	netx "github.com/2comjie/wali/core/net"
	"github.com/2comjie/wali/deploy"
	"github.com/2comjie/wali/locator"
	redisLocator "github.com/2comjie/wali/locator/redis"
	"github.com/2comjie/wali/network"
	redisRegistry "github.com/2comjie/wali/registry/redis"
)

var global *deploy.GateApp
var once sync.Once

func Init(options ...deploy.Option) {
	once.Do(func() {
		var err error
		options = append(options, deploy.WithServiceName(locator.GateName))
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

		// 2. 初始化日志/redis
		err = instruction.InitLogger(center)
		if err != nil {
			panic(err)
		}
		err = external.InitRedis(center)

		// 3. 初始化 注册中心/服务发现/locator/pprof
		options = append(options, deploy.WithRegistry(redisRegistry.NewRegistry(external.RedisRegistry())))
		options = append(options, deploy.WithDiscover(redisRegistry.NewDiscover(external.RedisRegistry())))
		options = append(options, deploy.WithLocator(redisLocator.NewProvider(external.RedisLocator())))
		options = append(options, deploy.WithComponents(pprof.StartPprof(flags.ServiceName, 0)))

		// 4. 初始化 rpc 服务
		privateIP, err := netx.PrivateIP()
		if err != nil {
			panic(err)
		}
		options = append(options, deploy.WithRPCHost(privateIP))

		// 5. 初始化网关 net 服务
		options = append(options, deploy.WithNetworkOptions(
			network.WithAuther(network.AuthFunc(func(token []byte) (uid string, err error) {
				jwt.Generate()
			})),

		))

		global, err = deploy.Gate(options...)
		if err != nil {
			panic(err)
		}
	})
}
