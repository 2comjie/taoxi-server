package nodeDeploy

import (
	"fmt"
	"sync"

	netx "github.com/2comjie/nova/core/net"
	"github.com/2comjie/nova/deploy"
	redisLocator "github.com/2comjie/nova/locator/redis"
	"github.com/2comjie/nova/logx"
	redisRegistry "github.com/2comjie/nova/registry/redis"
	"github.com/2comjie/taoxi-server/flags"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/2comjie/taoxi-server/internal/deploy/instruction"
	"github.com/2comjie/taoxi-server/pkg/pprof"
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
		options = append(options, deploy.WithRPCListen("0.0.0.0:0"), deploy.WithRPCHost(privateIP))

		global, err = deploy.Node(options...)

		logx.Infof("节点启动 %+v", global.Instance())
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
