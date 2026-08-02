package login

import (
	"fmt"

	"github.com/2comjie/taoxi-server/app/Api/login/internal/router"
	loginService "github.com/2comjie/taoxi-server/app/Api/login/internal/service"
	loginStore "github.com/2comjie/taoxi-server/app/Api/login/internal/store"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	"github.com/2comjie/taoxi-server/pkg/jwt"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/wali/etc"
)

func Init(args modules.Modules) {
	privateKey, err := jwt.ParsePrivateKey(etc.String(jwt.PrivateKeyEnv, "6QNmaXJ+0nPGEg4oWdhLS7M78VfDPxHN3O9EQ5IQ8wAcweG00kzN+CETgaunSd9g39eMCQdeyMmLtmfk4gtbRw=="))
	if err != nil {
		panic(fmt.Errorf("login: 读取%s失败: %w", jwt.PrivateKeyEnv, err))
	}
	store := loginStore.New(external.MysqlUser())
	loginManager := loginService.NewManager(privateKey)
	loginManager.Register(loginService.NewGuestLoginProvider(store))
	router.Init(args, loginManager)
}
