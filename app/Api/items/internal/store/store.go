package store

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	itement "github.com/2comjie/taoxi-server/app/Api/items/internal/store/ent"
	"github.com/2comjie/taoxi-server/app/Api/items/internal/store/ent/item"
	itemTypes "github.com/2comjie/taoxi-server/app/Api/items/types"
	"github.com/2comjie/taoxi-server/internal/config/items"
	"github.com/2comjie/taoxi-server/internal/config/shared"
)

var EntClient *itement.Client

func Init(driver *entsql.Driver) {
	if driver == nil {
		panic("items store: Ent Driver不能为空")
	}
	EntClient = itement.NewClient(itement.Driver(driver))
	initCache()
}

func Migrate(ctx context.Context) error {
	if err := EntClient.Schema.Create(ctx); err != nil {
		return fmt.Errorf("item: 创建道具表失败: %w", err)
	}
	return nil
}

func GetItems(ctx context.Context, uid uint64) ([]*itement.Item, error) {
	now := time.Now().Unix()

	return EntClient.Item.Query().
		Where(
			item.UID(uid),
			notExpiredPredicate(now),
			availablePredicate(),
		).
		All(ctx)
}

// 未过期：永久道具或者尚未到期
func notExpiredPredicate(now int64) func(*entsql.Selector) {
	return func(s *entsql.Selector) {
		s.Where(
			entsql.Or(
				entsql.EQ(item.FieldExpireAtUnix, 0),
				entsql.GT(item.FieldExpireAtUnix, now),
			),
		)
	}
}

// 仍有可用数量
func availablePredicate() func(*entsql.Selector) {
	return func(s *entsql.Selector) {
		s.Where(entsql.ExprP("`count` > `used_count`"))
	}
}

func AddItem(ctx context.Context, uid uint64, stackMode items.StackMode, itemTypeId items.ItemTypeId, count int64, expireAt time.Time) (int64, error) {
	tx, err := EntClient.Tx(ctx)
	if err != nil {
		return 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	err = lockUID(ctx, tx, uid)
	if err != nil {
		return 0, err
	}

	var ret int64
	if stackMode == items.StackModeTime {
		ret, err = addItemTimeStackInTx(ctx, tx, uid, itemTypeId, count, expireAt)
	} else {
		ret, err = addItemUnlimitedInTx(ctx, tx, uid, itemTypeId, count, expireAt)
	}
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	committed = true
	return ret, nil
}

func lockUID(ctx context.Context, tx *itement.Tx, uid uint64) error {
	return tx.ItemUserLock.Create().
		SetID(uid).
		OnConflict().
		Ignore().
		Exec(ctx)
}

func addItemTimeStackInTx(ctx context.Context, tx *itement.Tx, uid uint64, itemTypeId items.ItemTypeId, count int64, expireAt time.Time) (int64, error) {
	if count <= 0 {
		return 0, nil
	}

	now := time.Now().Unix()

	expireAtUnix := int64(0)
	if !expireAt.IsZero() {
		expireAtUnix = expireAt.Unix()
	}

	// 永久道具不能按时间叠加，按普通数量叠加处理
	if expireAtUnix == 0 {
		return addItemUnlimitedInTx(
			ctx, tx, uid, itemTypeId, count, expireAt,
		)
	}

	unitDuration := expireAtUnix - now
	if unitDuration <= 0 {
		return 0, nil
	}

	rows, err := tx.Item.Query().
		Where(
			item.UID(uid),
			item.ItemIDEQ(int32(itemTypeId)),
			item.ExpireAtUnixGT(now),
		).
		Order(func(s *entsql.Selector) {
			s.OrderExpr(
				entsql.Expr(
					"(`expire_at_unix` = 0) ASC, `expire_at_unix` ASC",
				),
			)
		}).
		All(ctx)
	if err != nil {
		return 0, err
	}

	// 新发放的 count 份道具转换为有效时长
	totalDuration := unitDuration * count

	// 合并当前仍然有效的时间堆叠记录
	for _, row := range rows {
		available := row.Count - row.UsedCount
		if available <= 0 {
			continue
		}

		totalDuration += (row.ExpireAtUnix - now) * available
	}

	newExpireAtUnix := now + totalDuration

	if len(rows) == 0 {
		err = tx.Item.Create().
			SetUID(uid).
			SetItemID(int32(itemTypeId)).
			SetCount(1).
			SetUsedCount(0).
			SetExpireAtUnix(newExpireAtUnix).
			SetCreateAtUnix(now).
			SetUpdateAtUnix(now).
			Exec(ctx)
		if err != nil {
			return 0, err
		}

		return 1, nil
	}

	// 时间叠加模式最终只保留一条记录
	selected := rows[0]

	for _, row := range rows[1:] {
		if err = tx.Item.DeleteOneID(row.ID).Exec(ctx); err != nil {
			return 0, err
		}
	}

	err = tx.Item.UpdateOneID(selected.ID).
		SetCount(1).
		SetUsedCount(0).
		SetExpireAtUnix(newExpireAtUnix).
		SetUpdateAtUnix(now).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return 1, nil
}

func addItemUnlimitedInTx(ctx context.Context, tx *itement.Tx, uid uint64, itemTypeId items.ItemTypeId, count int64, expireAt time.Time) (int64, error) {
	if count <= 0 {
		return 0, nil
	}

	now := time.Now().Unix()
	expireAtUnix := int64(0)
	if !expireAt.IsZero() {
		expireAtUnix = expireAt.Unix()
	}

	existing, err := tx.Item.Query().
		Where(
			item.UID(uid),
			item.ItemIDEQ(int32(itemTypeId)),
			item.ExpireAtUnixEQ(expireAtUnix),
		).
		First(ctx)
	if err != nil && !itement.IsNotFound(err) {
		return 0, err
	}

	if existing == nil {
		err = tx.Item.Create().
			SetUID(uid).
			SetItemID(int32(itemTypeId)).
			SetCount(count).
			SetUsedCount(0).
			SetExpireAtUnix(expireAtUnix).
			SetCreateAtUnix(now).
			SetUpdateAtUnix(now).
			Exec(ctx)
		if err != nil {
			return 0, err
		}
		return count, nil
	}

	err = tx.Item.UpdateOneID(existing.ID).
		AddCount(count).
		SetUpdateAtUnix(now).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return existing.Count + count - existing.UsedCount, nil
}

func AddItems(ctx context.Context, uid uint64, nonce string, rewards []*shared.Reward) error {
	tx, err := EntClient.Tx(ctx)
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err = lockUID(ctx, tx, uid); err != nil {
		return err
	}

	// nonce 和奖励在同一个事务内提交。
	err = tx.ItemNonce.Create().
		SetUID(uid).
		SetNonce(nonce).
		SetOpType(itemTypes.OperationTypeGrant).
		Exec(ctx)
	if err != nil {
		// nonce 已存在，说明之前已经成功发放。
		if itement.IsConstraintError(err) {
			return nil
		}
		return err
	}

	now := time.Now()
	for _, reward := range rewards {
		if reward == nil {
			return fmt.Errorf("items: 奖励不能为空")
		}

		itemConfig := items.GetItem(reward.ItemTypeId)
		if itemConfig == nil {
			return fmt.Errorf("items: 道具配置不存在 item_type_id=%d", reward.ItemTypeId)
		}

		var expireAt time.Time
		if reward.ExpireTimeUnix > 0 {
			expireAt = time.Unix(reward.ExpireTimeUnix, 0)
		} else if reward.ExpireDurationSec > 0 {
			expireAt = now.Add(time.Duration(reward.ExpireDurationSec) * time.Second)
		}

		switch itemConfig.StackMode {
		case items.StackModeTime:
			_, err = addItemTimeStackInTx(ctx, tx, uid, reward.ItemTypeId, int64(reward.Count), expireAt)
		case items.StackModeCount:
			_, err = addItemUnlimitedInTx(ctx, tx, uid, reward.ItemTypeId, int64(reward.Count), expireAt)
		default:
			return fmt.Errorf("items: 未知堆叠类型 item_type_id=%d stack_mode=%d", reward.ItemTypeId, itemConfig.StackMode)
		}
		if err != nil {
			return fmt.Errorf("items: 发放道具失败 item_type_id=%d: %w", reward.ItemTypeId, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	committed = true
	return nil
}
