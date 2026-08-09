package store

import (
	"context"

	itemTypes "github.com/2comjie/taoxi-server/app/Api/items/types"
	"github.com/2comjie/taoxi-server/internal/config/items"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	itemRedisKey "github.com/2comjie/taoxi-server/internal/redis_key/item"
	"github.com/2comjie/taoxi-server/pkg/cachex"
)

var bagItemCache *cachex.VersionedCache[uint64, []itemTypes.Item]

func initCache() {
	bagItemCache = cachex.NewVersionedCache[uint64, []itemTypes.Item](external.RedisGame(), func(u uint64) string {
		return itemRedisKey.UserItemVersion(u)
	}, func(ctx context.Context, key uint64) ([]itemTypes.Item, error) {
		itemRows, err := GetItems(ctx, key)
		if err != nil {
			return nil, err
		}

		entries := make([]itemTypes.Item, 0, len(itemRows))
		for _, item := range itemRows {
			entries = append(entries, itemTypes.Item{
				BagId:        item.ID,
				ItemTypeId:   items.ItemTypeId(item.ItemID),
				Quantity:     item.Count - item.UsedCount,
				ExpireAtUnix: item.ExpireAtUnix,
			})
		}
		return entries, nil
	})
}

func GetItemWithCache(ctx context.Context, uid uint64) ([]itemTypes.Item, error) {
	return bagItemCache.Get(ctx, uid)
}

func InvalidateItemCache(ctx context.Context, uid uint64) error {
	return bagItemCache.Invalidate(ctx, uid)
}
