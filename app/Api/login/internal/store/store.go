package loginStore

import (
	"context"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	loginent "github.com/2comjie/taoxi-server/app/Api/login/internal/store/ent"
	"github.com/2comjie/taoxi-server/app/Api/login/internal/store/ent/identity"
	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	"github.com/2comjie/wali/logx"
)

var ErrAccountDeleted = errors.New("login: 账号已注销")

type Store struct {
	client *loginent.Client
}

func New(driver *entsql.Driver) *Store {
	return &Store{
		client: loginent.NewClient(loginent.Driver(driver)),
	}
}

func (s *Store) Migrate(ctx context.Context) error {
	err := s.client.Schema.Create(ctx)
	if err != nil {
		return fmt.Errorf("login: 创建账号表失败: %w", err)
	}
	return nil
}

func (s *Store) FindOrCrateAccount(ctx context.Context, loginType loginTypes.LoginType, loginIdentity loginTypes.Identity) (uint64, bool, error) {
	logCtx := logx.WithField("action", "查找或者创建账号").WithField("loginType", loginType).WithField("openId", loginIdentity.OpenID)
	logCtx.Infof("查询第三方登录记录")
	uid, found, err := s.findIdentity(ctx, loginType, loginIdentity.AppID, loginIdentity.OpenID)
	if err != nil {
		logCtx.Errorf("查询登陆记录失败: %v", err)
		return 0, false, err
	}
	logCtx.Infof("登陆记录: %v %v", uid, found)
	if found {
		logCtx.Infof("有登陆记录 查找内部账号")
		if err = s.checkAccount(ctx, uid); err != nil {
			logCtx.Errorf("查询内部账号失败: %v", err)
			return 0, false, err
		}
		logCtx.Infof("已经注册")
		return uid, false, nil
	}

	logCtx.Infof("没有登陆记录 创建内部账号和登陆记录")
	uid, err = s.createAccount(ctx, loginType, loginIdentity)
	if err == nil {
		logCtx.Infof("创建内部账号成功 uid %d", uid)
		return uid, true, nil
	}
	if !loginent.IsConstraintError(err) {
		logCtx.Errorf("创建内部账号失败: %v", err)
		return 0, false, err
	}

	// 多个API节点可能同时为同一个第三方身份注册 唯一索引冲突的一方
	// 回滚自己创建的账号，再读取成功一方已经提交的UID
	uid, found, findErr := s.findIdentity(ctx, loginType, loginIdentity.AppID, loginIdentity.OpenID)
	if findErr != nil {
		return 0, false, findErr
	}
	if !found {
		return 0, false, err
	}
	if checkErr := s.checkAccount(ctx, uid); checkErr != nil {
		return 0, false, checkErr
	}
	logCtx.Infof("并发注册命中已有账号 uid %d", uid)
	return uid, false, nil
}

func (s *Store) findIdentity(ctx context.Context, loginType loginTypes.LoginType, appID, openID string) (uint64, bool, error) {
	only, err := s.client.Identity.Query().Where(
		identity.LoginTypeEQ(int32(loginType)),
		identity.AppIDEQ(appID),
		identity.OpenIDEQ(openID),
	).Only(ctx)
	if loginent.IsNotFound(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("login: 查询登录身份失败: %w", err)
	}
	return only.UID, true, nil
}

func (s *Store) checkAccount(ctx context.Context, uid uint64) error {
	accountData, err := s.client.Account.Get(ctx, uid)
	if err != nil {
		if loginent.IsNotFound(err) {
			return fmt.Errorf("login: 登录身份对应的账号不存在 uid=%d", uid)
		}
		return fmt.Errorf("login: 查询账号失败: %w", err)
	}
	if accountData.IsDeleted {
		return ErrAccountDeleted
	}
	return nil
}

func (s *Store) createAccount(ctx context.Context, loginType loginTypes.LoginType, loginIdentity loginTypes.Identity) (uint64, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return 0, fmt.Errorf("login: 开启创建账号事务失败: %w", err)
	}
	rollback := func() {
		_ = tx.Rollback()
	}

	accountData, err := tx.Account.Create().Save(ctx)
	if err != nil {
		rollback()
		return 0, fmt.Errorf("login: 创建账号失败: %w", err)
	}

	createIdentity := tx.Identity.Create().
		SetUID(accountData.ID).
		SetLoginType(int32(loginType)).
		SetAppID(loginIdentity.AppID).
		SetOpenID(loginIdentity.OpenID)
	if loginIdentity.UnionID != "" {
		createIdentity.SetUnionID(loginIdentity.UnionID)
	}
	if _, err = createIdentity.Save(ctx); err != nil {
		rollback()
		return 0, fmt.Errorf("login: 创建登陆记录失败: %w", err)
	}

	if err = tx.Commit(); err != nil {
		rollback()
		return 0, fmt.Errorf("login: 提交创建账号事务失败: %w", err)
	}
	return accountData.ID, nil
}
