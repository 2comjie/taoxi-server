package login

import (
	"crypto/ed25519"
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
	publicKey := privateKey.Public().(ed25519.PublicKey)
	if err = jwt.InitPublicKey(publicKey); err != nil {
		panic(fmt.Errorf("login: 初始化JWT验签公钥失败: %w", err))
	}
	loginManager := loginService.NewManager(store, privateKey)
	loginManager.Register(loginService.NewGuestLoginProvider(store))
	router.Init(args, loginManager)
}
