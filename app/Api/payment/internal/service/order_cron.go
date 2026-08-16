package paymentService

import (
	"context"
	"time"

	redisLock "github.com/2comjie/nova/lock/redis"
	"github.com/2comjie/nova/logx"
	"github.com/2comjie/nova/logx/logdef"
	paymentStore "github.com/2comjie/taoxi-server/app/Api/payment/internal/store"
	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	paymentRedisKey "github.com/2comjie/taoxi-server/internal/redis_key/payment"
	"github.com/2comjie/taoxi-server/pkg/stderr"
)

const (
	pendingOrderTimeout = 15 * time.Minute
	expiredOrderBatch   = 100
	unlockTimeout       = 3 * time.Second
)

func CheckExpiredPendingOrders(ctx context.Context) *stderr.Error {
	logCtx := logx.WithField("action", "扫描超时支付订单")
	lease, locked, err := redisLock.TryLockDefault(redisClient, ctx, paymentRedisKey.TimeoutScanLock())
	if err != nil {
		logCtx.Errorf("获取超时订单扫描锁失败 err=%v", err)
		return stderr.InternalServerError("获取超时订单扫描锁失败")
	}
	if !locked {
		return nil
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), unlockTimeout)
		defer cancel()
		if err := lease.Unlock(unlockCtx); err != nil {
			logx.Errorf("payment: 释放超时订单扫描锁失败 err=%v", err)
		}
	}()

	expiredAtUnix := time.Now().Add(-pendingOrderTimeout).Unix()
	orders, err := paymentStore.FindExpiredPendingOrders(ctx, expiredAtUnix, expiredOrderBatch)
	if err != nil {
		logCtx.Errorf("查询超时支付订单失败 err=%v", err)
		return stderr.InternalServerError("查询超时支付订单失败")
	}

	for _, order := range orders {
		if err := ctx.Err(); err != nil {
			logCtx.Errorf("扫描超时支付订单中断 err=%v", err)
			return stderr.InternalServerError("扫描超时支付订单中断")
		}
		if err := lease.Err(); err != nil {
			logCtx.Errorf("超时订单扫描锁失效 err=%v", err)
			return stderr.InternalServerError("超时订单扫描锁失效")
		}
		processExpiredPendingOrder(ctx, order, expiredAtUnix)
	}
	return nil
}

func processExpiredPendingOrder(ctx context.Context, order *paymentent.PaymentOrder, expiredAtUnix int64) {
	logCtx := logx.WithField("action", "处理超时支付订单").WithField("order_id", order.ID).WithField("uid", order.UID)

	addRetryTimes := func(logCtx logdef.ILogger, ctx context.Context, orderId uint64) {
		if err := paymentStore.AddRetryTimes(ctx, orderId); err != nil {
			logCtx.Errorf("增加订单重试次数失败 err=%v", err)
		}
	}

	lease, locked, err := redisLock.TryLockDefault(redisClient, ctx, paymentRedisKey.UserLock(order.UID))
	if err != nil {
		logCtx.Errorf("获取玩家支付锁失败 err=%v", err)
		return
	}
	if !locked {
		return
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), unlockTimeout)
		defer cancel()
		if err := lease.Unlock(unlockCtx); err != nil {
			logCtx.Errorf("释放玩家支付锁失败 err=%v", err)
		}
	}()

	current, err := paymentStore.GetExpiredPendingOrder(ctx, order.ID, expiredAtUnix)
	if err != nil {
		logCtx.Errorf("重新查询超时订单失败 err=%v", err)
		return
	}
	if current == nil {
		return
	}

	if current.Credential == nil || *current.Credential == "" {
		if err := paymentStore.MarkOrderTimeout(ctx, current.ID, expiredAtUnix); err != nil {
			logCtx.Errorf("关闭无凭证超时订单失败 err=%v", err)
			return
		}
		logCtx.Infof("无支付凭证的Pending订单已超时关闭")
		return
	}

	channel, found := GetChannel(current.PaymentType)
	if !found {
		logCtx.Errorf("支付渠道未注册 payment_type=%d", current.PaymentType)
		addRetryTimes(logCtx, ctx, current.ID)
		return
	}

	credential := *current.Credential
	thirdPartyOrder, err := channel.QueryOrder(ctx, credential)
	if err != nil {
		logCtx.Errorf("查询第三方订单失败 err=%v", err)
		addRetryTimes(logCtx, ctx, current.ID)
		return
	}
	if thirdPartyOrder == nil {
		logCtx.Errorf("第三方订单不存在")
		addRetryTimes(logCtx, ctx, current.ID)
		return
	}

	current, err = channel.FindOrBind(ctx, current.UID, credential, thirdPartyOrder)
	if err != nil {
		logCtx.Errorf("查找或绑定第三方订单失败 err=%v", err)
		addRetryTimes(logCtx, ctx, order.ID)
		return
	}
	if current == nil {
		logCtx.Errorf("未找到对应的内部订单")
		addRetryTimes(logCtx, ctx, order.ID)
		return
	}
	if thirdPartyOrder.ProductId == "" || thirdPartyOrder.ProductId != current.ThirdPartyProductID {
		logCtx.Errorf("第三方商品不匹配 expected=%s actual=%s", current.ThirdPartyProductID, thirdPartyOrder.ProductId)
		addRetryTimes(logCtx, ctx, current.ID)
		return
	}

	result, stdErr := ProcessOrderWithoutLock(logCtx, ctx, channel, current, thirdPartyOrder)
	if stdErr != nil {
		logCtx.Errorf("处理第三方订单状态失败 code=%d msg=%s", stdErr.Code, stdErr.Msg)
		addRetryTimes(logCtx, ctx, current.ID)
		return
	}
	if result != paymentTypes.OrderProcessWaiting {
		return
	}

	if err := paymentStore.MarkOrderTimeout(ctx, current.ID, expiredAtUnix); err != nil {
		logCtx.Errorf("关闭超时Pending订单失败 err=%v", err)
		return
	}
	logCtx.Infof("第三方订单仍为Pending，内部订单已超时关闭")
}
