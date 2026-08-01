package store

import (
	"context"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/2comjie/taoxi-server/app/Api/login/internal/store/ent/playeridentity"

	loginent "github.com/2comjie/taoxi-server/app/Api/login/internal/store/ent"
	logintypes "github.com/2comjie/taoxi-server/app/Api/login/types"
)

const playerStatusActive int8 = 1

var ErrPlayerDisabled = errors.New("login: 玩家账号不可用")

type Store struct {
	client *loginent.Client
	newUID func() string
}

func New(driver *entsql.Driver, newUID func() string) (*Store, error) {
	if driver == nil {
		return nil, errors.New("login: Ent driver不能为空")
	}
	if newUID == nil {
		return nil, errors.New("login: UID生成器不能为空")
	}
	return &Store{
		client: loginent.NewClient(loginent.Driver(driver)),
		newUID: newUID,
	}, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	if err := s.client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("login: 创建账号表失败: %w", err)
	}
	return nil
}

func (s *Store) FindOrCreatePlayer(ctx context.Context, loginType logintypes.LoginType, identity logintypes.Identity) (string, bool, error) {
	uid, found, err := s.findIdentity(ctx, loginType, identity.AppID, identity.OpenID)
	if err != nil {
		return "", false, err
	}
	if found {
		if err := s.checkPlayer(ctx, uid); err != nil {
			return "", false, err
		}
		return uid, false, nil
	}

	uid = s.newUID()
	if uid == "" || len(uid) > 32 {
		return "", false, errors.New("login: UID生成失败")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return "", false, fmt.Errorf("login: 开启账号事务失败: %w", err)
	}
	rollback := func() {
		_ = tx.Rollback()
	}

	if _, err = tx.Player.Create().SetID(uid).SetStatus(playerStatusActive).Save(ctx); err != nil {
		rollback()
		return "", false, fmt.Errorf("login: 创建玩家失败: %w", err)
	}
	createIdentity := tx.PlayerIdentity.Create().
		SetUID(uid).
		SetLoginType(int32(loginType)).
		SetAppID(identity.AppID).
		SetOpenID(identity.OpenID)
	if identity.UnionID != "" {
		createIdentity.SetUnionID(identity.UnionID)
	}
	if _, err = createIdentity.Save(ctx); err != nil {
		rollback()
		if loginent.IsConstraintError(err) {
			winnerUID, winnerFound, findErr := s.findIdentity(ctx, loginType, identity.AppID, identity.OpenID)
			if findErr != nil {
				return "", false, findErr
			}
			if winnerFound {
				if err := s.checkPlayer(ctx, winnerUID); err != nil {
					return "", false, err
				}
				return winnerUID, false, nil
			}
		}
		return "", false, fmt.Errorf("login: 绑定登录身份失败: %w", err)
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
		return "", false, fmt.Errorf("login: 提交账号事务失败: %w", err)
	}
	return uid, true, nil
}

func (s *Store) findIdentity(ctx context.Context, loginType logintypes.LoginType, appID, openID string) (string, bool, error) {
	identity, err := s.client.PlayerIdentity.Query().Where(
		playeridentity.LoginTypeEQ(int32(loginType)),
		playeridentity.AppIDEQ(appID),
		playeridentity.OpenIDEQ(openID),
	).Only(ctx)
	if loginent.IsNotFound(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("login: 查询登录身份失败: %w", err)
	}
	return identity.UID, true, nil
}

func (s *Store) checkPlayer(ctx context.Context, uid string) error {
	player, err := s.client.Player.Get(ctx, uid)
	if loginent.IsNotFound(err) {
		return errors.New("login: 登录身份对应的玩家不存在")
	}
	if err != nil {
		return fmt.Errorf("login: 查询玩家失败: %w", err)
	}
	if player.Status != playerStatusActive {
		return ErrPlayerDisabled
	}
	return nil
}
