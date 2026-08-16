package main

import (
	"github.com/2comjie/nova/deploy"
	gateRouter "github.com/2comjie/taoxi-server/app/Gate/router"
	gateDeploy "github.com/2comjie/taoxi-server/internal/deploy/gate"
)

func main() {
	gateDeploy.Init(deploy.WithGateRouter(gateRouter.Init()))
}
