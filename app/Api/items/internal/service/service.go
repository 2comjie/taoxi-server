package service

import (
	"context"
	"time"

	itemStore "github.com/2comjie/taoxi-server/app/Api/items/internal/store"
	itemTypes "github.com/2comjie/taoxi-server/app/Api/items/types"
	"github.com/2comjie/taoxi-server/internal/config/items"
	"github.com/2comjie/taoxi-server/internal/config/shared"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx"
)

func AddItem(ctx context.Context, uid uint64, stackMode items.StackMode, itemTypeId items.ItemTypeId, count int64, expireAt time.Time) (int64, *stderr.Error) {
	ret, err := itemStore.AddItem(ctx, uid, stackMode, itemTypeId, count, expireAt)
	if err != nil {
		logx.Errorf("items: 添加道具失败 uid=%d item_type_id=%d count=%d err=%v", uid, itemTypeId, count, err)
		return 0, stderr.InternalServerError("添加道具失败")
	}

	// 数据库事务已经提交。缓存失效失败不能让调用方认为发放失败，
	// 否则调用方重试会导致同一份道具被重复发放
	if err = itemStore.InvalidateItemCache(ctx, uid); err != nil {
		logx.Errorf("items: 背包缓存失效失败 uid=%d err=%v", uid, err)
	}

	return ret, nil
}

func GetItemWithCache(ctx context.Context, uid uint64) ([]itemTypes.Item, *stderr.Error) {
	cache, err := itemStore.GetItemWithCache(ctx, uid)
	if err != nil {
		logx.Errorf("items: 获取背包缓存失败 uid=%d err=%v", uid, err)
		return nil, stderr.InternalServerError("获取背包缓存失败")
	}
	return cache, nil
}

func AddItems(ctx context.Context, uid uint64, nonce string, rewards []*shared.Reward) *stderr.Error {
	err := itemStore.AddItems(ctx, uid, nonce, rewards)
	if err != nil {
		logx.Errorf("发放道具失败 uid=%d nonce=%s err=%v", uid, nonce, err)
		return stderr.InternalServerError("发放道具失败")
	}

	err = itemStore.InvalidateItemCache(ctx, uid)
	if err != nil {
		logx.Errorf("items: 背包缓存失效失败 uid=%d err=%v", uid, err)
	}
	return nil
}
