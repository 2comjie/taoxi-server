package login

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/2comjie/taoxi-server/app/Api/login/internal/router"
	"github.com/2comjie/taoxi-server/app/Api/login/internal/service"
	"github.com/2comjie/taoxi-server/app/Api/login/internal/store"
	"github.com/2comjie/taoxi-server/pkg/jwt"
	"github.com/2comjie/taoxi-server/pkg/middleware/auth"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/wali/etc"
)

const (
	googleClientIDEnv  = "TAOXI_GOOGLE_CLIENT_ID"
	googleSecretEnv    = "TAOXI_GOOGLE_CLIENT_SECRET"
	googleRedirectEnv  = "TAOXI_GOOGLE_REDIRECT_URL"
	appleAudiencesEnv  = "TAOXI_APPLE_AUDIENCES"
	weChatAppIDEnv     = "TAOXI_WECHAT_APP_ID"
	weChatAppSecretEnv = "TAOXI_WECHAT_APP_SECRET"
)

func Init(args modules.Modules, driver *entsql.Driver, newUID func() string, autoMigrate bool) error {
	privateKey, err := jwt.ParsePrivateKey(etc.String(jwt.PrivateKeyEnv, "6QNmaXJ+0nPGEg4oWdhLS7M78VfDPxHN3O9EQ5IQ8wAcweG00kzN+CETgaunSd9g39eMCQdeyMmLtmfk4gtbRw=="))
	if err != nil {
		return fmt.Errorf("login: 读取%s失败: %w", jwt.PrivateKeyEnv, err)
	}
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return errors.New("login: 无法从JWT私钥获取公钥")
	}

	accountStore, err := store.New(driver, newUID)
	if err != nil {
		return err
	}
	if autoMigrate {
		err = accountStore.Migrate(context.Background())
		if err != nil {
			return err
		}
	}

	manager, err := service.NewManager(accountStore, privateKey)
	if err != nil {
		return err
	}
	err = manager.RegisterProvider(service.GuestProvider{})
	if err != nil {
		return err
	}

	args.ClientGroup.Use(auth.AccessToken(publicKey))
	router.Init(args, manager)
	return nil
}
