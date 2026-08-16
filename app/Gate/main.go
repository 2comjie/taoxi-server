package main

import (
	gateRouter "github.com/2comjie/taoxi-server/app/Gate/router"
	gateDeploy "github.com/2comjie/taoxi-server/internal/deploy/gate"
	"github.com/2comjie/wali/deploy"
)

func main() {
	gateDeploy.Init(deploy.WithGateRouter(gateRouter.Init()))
}
