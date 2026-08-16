package limit

import (
	"context"
	"net/http"
	"time"

	redisLock "github.com/2comjie/nova/lock/redis"
	"github.com/2comjie/nova/logx"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/xhttp"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
)

var cc = cache.New(time.Minute, time.Minute)

// Limiter 限制每sec秒，执行limitType类型的请求不超过limit次
// limit:最大请求数  sec:时间限制  limitType:限制类型
func Limiter(limitType string, limit, sec int32) gin.HandlerFunc {
	return func(c *gin.Context) {
		request, _ := midef.GetClientRequestHeader(c)
		if request == nil {
			xhttp.Fail(c, http.StatusBadRequest, "请求参数错误", "")
			c.Abort()
			return
		}
		target := request.DeviceID
		if len(target) == 0 {
			target = c.ClientIP()
		}
		key := limitType + "." + target
		if v, ok := cc.Get(key); ok {
			count := v.(int32)
			if count >= limit {
				logx.WithField("key", key).Info("请求频率限制")
				xhttp.Fail(c, http.StatusTooManyRequests, "请求频繁", "")
				c.Abort()
				return
			}
			if _, err := cc.IncrementInt32(key, 1); err != nil {
				logx.WithError(err).WithField("key", key).Warn("Limiter 计数递增失败")
				xhttp.Fail(c, http.StatusTooManyRequests, "请求频繁", "")
				c.Abort()
				return
			}
		} else {
			cc.Set(key, int32(1), time.Second*time.Duration(sec))
		}
		c.Next()
	}
}

func LockUser(keyFn func(uid uint64) string, rc redis.UniversalClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		request, _ := midef.GetClientRequestHeader(c)
		if request == nil {
			xhttp.Fail(c, http.StatusBadRequest, "请求参数错误", "")
			c.Abort()
			return
		}
		target := request.Uid
		lockKey := keyFn(target)
		lease, locked, err := redisLock.TryLockDefault(rc, context.Background(), lockKey)
		if err != nil {
			logx.WithError(err).WithField("lockKey", lockKey).Warn("LockUser 获取锁失败")
			xhttp.Fail(c, http.StatusInternalServerError, "系统错误", "")
			c.Abort()
			return
		}
		if !locked {
			logx.WithField("lockKey", lockKey).Info("LockUser 锁被占用")
			xhttp.Fail(c, http.StatusTooManyRequests, "请求频繁", "")
			c.Abort()
			return
		}
		defer func() {
			logx.Infof("LockUser 锁释放 %s", lockKey)
			unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if unlockErr := lease.Unlock(unlockCtx); unlockErr != nil {
				logx.Errorf("释放用户锁失败 err %v", unlockErr)
			}
		}()
		c.Next()
	}
}
