package store

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	itement "github.com/2comjie/taoxi-server/app/Api/items/internal/store/ent"
	"github.com/2comjie/taoxi-server/app/Api/items/internal/store/ent/item"
	sharedItem "github.com/2comjie/taoxi-server/internal/shared/item"
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

func AddItem(ctx context.Context, uid uint64, stackMode sharedItem.StackMode, itemTypeId int32, count int64, expireAt time.Time) (int64, error) {
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
	if stackMode == sharedItem.StackModeTime {
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

func addItemTimeStackInTx(ctx context.Context, tx *itement.Tx, uid uint64, itemTypeId int32, count int64, expireAt time.Time) (int64, error) {
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
			item.ItemIDEQ(itemTypeId),
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
			SetItemID(itemTypeId).
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

func addItemUnlimitedInTx(ctx context.Context, tx *itement.Tx, uid uint64, itemTypeId int32, count int64, expireAt time.Time) (int64, error) {
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
			item.ItemIDEQ(itemTypeId),
			item.ExpireAtUnixEQ(expireAtUnix),
		).
		First(ctx)
	if err != nil && !itement.IsNotFound(err) {
		return 0, err
	}

	if existing == nil {
		err = tx.Item.Create().
			SetUID(uid).
			SetItemID(itemTypeId).
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
