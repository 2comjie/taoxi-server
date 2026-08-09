package cachex

import (
	"context"
	"errors"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"golang.org/x/sync/singleflight"
)

type KeyFunc[K ristretto.Key] func(K) string

type LoadFunc[K ristretto.Key, T any] func(ctx context.Context, key K) (T, error)

type versionedEntry[T any] struct {
	Data    T
	Version int64
}

type VersionedCache[K ristretto.Key, T any] struct {
	redis redis.UniversalClient

	keyFunc  KeyFunc[K]
	loadFunc LoadFunc[K, T]

	local *ristretto.Cache[K, versionedEntry[T]]
	group singleflight.Group

	localTTL   time.Duration
	versionTTL time.Duration
}

type VersionedCacheOption struct {
	MaxKeys    int64
	LocalTTL   time.Duration
	VersionTTL time.Duration
}

func DefaultVersionedCacheOption() VersionedCacheOption {
	return VersionedCacheOption{
		MaxKeys:    5000,
		LocalTTL:   5 * time.Minute,
		VersionTTL: 24 * time.Hour,
	}
}

func NewVersionedCache[K ristretto.Key, T any](rc redis.UniversalClient, keyFunc KeyFunc[K], loadFunc LoadFunc[K, T], options ...VersionedCacheOption) *VersionedCache[K, T] {
	if rc == nil {
		panic("cachex: Redis客户端不能为空")
	}
	if keyFunc == nil {
		panic("cachex: KeyFunc不能为空")
	}
	if loadFunc == nil {
		panic("cachex: LoadFunc不能为空")
	}

	option := DefaultVersionedCacheOption()
	if len(options) > 0 {
		option = options[0]
	}
	if option.MaxKeys <= 0 {
		option.MaxKeys = 5000
	}
	if option.LocalTTL <= 0 {
		option.LocalTTL = 5 * time.Minute
	}

	local, err := ristretto.NewCache(
		&ristretto.Config[K, versionedEntry[T]]{
			NumCounters: option.MaxKeys * 10,
			MaxCost:     option.MaxKeys,
			BufferItems: 64,
		},
	)
	if err != nil {
		panic("cachex: 创建本地缓存失败: " + err.Error())
	}

	return &VersionedCache[K, T]{
		redis:      rc,
		keyFunc:    keyFunc,
		loadFunc:   loadFunc,
		local:      local,
		localTTL:   option.LocalTTL,
		versionTTL: option.VersionTTL,
	}
}

func (c *VersionedCache[K, T]) Get(ctx context.Context, key K) (T, error) {
	redisKey := c.keyFunc(key)

	remoteVersion, redisErr := c.getRemoteVersion(ctx, redisKey)
	localEntry, hasLocal := c.local.Get(key)

	// Redis异常时允许使用本地旧数据降级
	if redisErr != nil && hasLocal {
		return localEntry.Data, nil
	}

	// 本地版本与Redis版本一致
	if redisErr == nil &&
		hasLocal &&
		localEntry.Version == remoteVersion {
		return localEntry.Data, nil
	}

	value, err, _ := c.group.Do(redisKey, func() (any, error) {
		// 等待singleflight期间，其他请求可能已经完成回源
		latestVersion, versionErr := c.getRemoteVersion(ctx, redisKey)
		currentEntry, exists := c.local.Get(key)

		if versionErr == nil &&
			exists &&
			currentEntry.Version == latestVersion {
			return currentEntry, nil
		}

		data, loadErr := c.loadFunc(ctx, key)
		if loadErr != nil {
			if exists {
				return currentEntry, nil
			}
			return nil, loadErr
		}

		if versionErr != nil {
			latestVersion = 0
		}

		entry := versionedEntry[T]{
			Data:    data,
			Version: latestVersion,
		}
		c.local.SetWithTTL(key, entry, 1, c.localTTL)
		return entry, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}

	return value.(versionedEntry[T]).Data, nil
}

func (c *VersionedCache[K, T]) Invalidate(ctx context.Context, key K) error {
	// 当前节点立即删除，保证本节点写后读。
	c.local.Del(key)

	redisKey := c.keyFunc(key)

	if c.versionTTL <= 0 {
		return c.redis.Incr(ctx, redisKey).Err()
	}

	pipe := c.redis.TxPipeline()
	pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, c.versionTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *VersionedCache[K, T]) getRemoteVersion(ctx context.Context, redisKey string) (int64, error) {
	value, err := c.redis.Get(ctx, redisKey).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return cast.ToInt64(value), nil
}
