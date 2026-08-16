package main

import (
	"github.com/2comjie/taoxi-server/app/Player/service"
	nodeDeploy "github.com/2comjie/taoxi-server/internal/deploy/node"
	"github.com/2comjie/wali/deploy"
	"google.golang.org/grpc"
)

func main() {
	rpcServer := grpc.NewServer()
	nodeDeploy.Init(deploy.WithRPCServer(rpcServer))
	service.Init(nodeDeploy.App(), rpcServer)

	if err := nodeDeploy.App().Run(); err != nil {
		panic(err)
	}
}
