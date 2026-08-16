package gateDeploy

import (
	"fmt"
	"sync"

	"github.com/2comjie/nova/app/gate"
	netx "github.com/2comjie/nova/core/net"
	"github.com/2comjie/nova/core/zipper"
	"github.com/2comjie/nova/deploy"
	"github.com/2comjie/nova/etc"
	"github.com/2comjie/nova/locator"
	redisLocator "github.com/2comjie/nova/locator/redis"
	"github.com/2comjie/nova/logx"
	"github.com/2comjie/nova/network"
	netKcp "github.com/2comjie/nova/network/transport/kcp"
	netTcp "github.com/2comjie/nova/network/transport/tcp"
	netWs "github.com/2comjie/nova/network/transport/ws"
	redisRegistry "github.com/2comjie/nova/registry/redis"
	"github.com/2comjie/taoxi-server/flags"
	gateConfig "github.com/2comjie/taoxi-server/internal/config/gate"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/2comjie/taoxi-server/internal/deploy/instruction"
	"github.com/2comjie/taoxi-server/pkg/jwt"
	"github.com/2comjie/taoxi-server/pkg/pprof"
)

var global *deploy.GateApp
var once sync.Once

const (
	MetaTCPExternalAddress = "gate.tcp.external_address"
	MetaKCPExternalAddress = "gate.kcp.external_address"
	MetaWSExternalAddress  = "gate.ws.external_address"
)

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
		if err = gateConfig.Init(center); err != nil {
			panic(err)
		}
		networkConfig, err := gateConfig.GetNetwork(flags.ServiceIndex)
		if err != nil {
			panic(err)
		}

		// 2. 初始化日志/redis
		err = instruction.InitLogger(center)
		if err != nil {
			panic(err)
		}
		err = external.InitRedis(center)
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

		// 5. 初始化网关 net 服务
		tcpListener, err := netTcp.Listen(networkConfig.TCPListenAddress)
		if err != nil {
			panic(err)
		}
		kcpListener, err := netKcp.Listen(networkConfig.KCPListenAddress)
		if err != nil {
			panic(err)
		}
		wsListener, err := netWs.Listen(networkConfig.WSListenAddress)
		if err != nil {
			panic(err)
		}

		publicKey, err := jwt.ParsePublicKey(etc.String(jwt.PublicKeyEnv, "HMHhtNJMzfghE4Grp0nfYN/XjAkHXsjJi7Zn5OILW0c="))
		if err != nil {
			panic(fmt.Errorf("gateDeploy: 读取%s失败: %w", jwt.PublicKeyEnv, err))
		}
		err = jwt.InitPublicKey(publicKey)
		if err != nil {
			panic(fmt.Errorf("gateDeploy: 初始化JWT验签公钥失败: %w", err))
		}
		options = append(options, deploy.WithNetworkOptions(
			network.WithAuther(network.AuthFunc(func(token []byte) (uid string, err error) {
				uid, err = jwt.Auth(token)
				if err != nil {
					return "", err
				}
				return uid, nil
			})),
			network.WithListener(tcpListener),
			network.WithListener(kcpListener),
			network.WithListener(wsListener),
			network.WithZipper(zipper.NewSnappy()),
		),
		)
		options = append(options, deploy.WithGateErrorHandler(func(ctx *gate.Context, err error) {
			logx.Warnf("gate error: %+v ctx: %+v", err, ctx)
		}))
		options = append(options, deploy.WithGateHooks(network.Hooks{
			OnSessionStart: func(session *network.Session) {
				logx.Infof("gate session start: %+v", session)
			},
			OnSessionEnd: func(session *network.Session) {
				logx.Infof("gate session end: %+v", session)
			},
			OnSessionBind: func(session *network.Session) error {
				logx.Infof("gate session bind: %+v", session)
				return nil
			},
			OnHeartbeat: func(session *network.Session) {
				logx.Debugf("gate session heartbeat: %+v", session)
			},
			OnReq: func(context *network.ReqContext) {
				logx.Debugf("gate req: %+v", context)
			},
		}))

		options = append(options, deploy.WithMetaData(map[string]string{
			MetaTCPExternalAddress: networkConfig.TCPExternalAddress,
			MetaKCPExternalAddress: networkConfig.KCPExternalAddress,
			MetaWSExternalAddress:  networkConfig.WSExternalAddress,
		}))

		global, err = deploy.Gate(options...)
		if err != nil {
			panic(err)
		}

		logx.Infof("网关启动 %+v", global.Instance())
		err = global.Run()
		if err != nil {
			panic(err)
		}
	})
}
