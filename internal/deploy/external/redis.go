package external

import (
	"context"
	"fmt"

	"github.com/2comjie/nova/config"
	"github.com/redis/go-redis/v9"
)

var redisMap = make(map[string]redis.UniversalClient)

func InitRedis(center config.Config) error {
	configMap := map[string]*redis.UniversalOptions{}
	redisConfig := center.Value("bootstrap.redis")
	if redisConfig == nil {
		return fmt.Errorf("nodeDeploy: 缺少 redis 配置")
	}
	err := redisConfig.Scan(&configMap)
	if err != nil {
		return fmt.Errorf("nodeDeploy: 解析 redis 配置失败: %w", err)
	}
	for name, options := range configMap {
		redisClient := newRedisClient(options)
		redisMap[name] = redisClient
	}

	for _, redisClient := range redisMap {
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			return fmt.Errorf("nodeDeploy: 连接 redis 失败: %w", err)
		}
	}
	return nil
}

func newRedisClient(opts *redis.UniversalOptions) redis.UniversalClient {
	buffSize := 1024 * 10
	switch {
	case len(opts.Addrs) > 1 || opts.IsClusterMode:
		options := opts.Cluster()
		options.ReadBufferSize = buffSize
		options.WriteBufferSize = buffSize
		return redis.NewClusterClient(options)
	default:
		options := opts.Simple()
		options.ReadBufferSize = buffSize
		options.WriteBufferSize = buffSize
		return redis.NewClient(options)
	}
}

func GetRedisClient(name string) redis.UniversalClient {
	return redisMap[name]
}

func RedisGame() redis.UniversalClient {
	return redisMap["game"]
}
func RedisUser() redis.UniversalClient {
	return redisMap["user"]
}
func RedisPayment() redis.UniversalClient {
	return redisMap["payment"]
}
func RedisRegistry() redis.UniversalClient {
	return redisMap["registry"]
}
func RedisLocator() redis.UniversalClient {
	return redisMap["locator"]
}
func RedisAsynq() redis.UniversalClient {
	return redisMap["asynq"]
}
