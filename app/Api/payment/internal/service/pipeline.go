package paymentService

import (
	"context"
	"encoding/json"
	paymentConfig "github.com/2comjie/taoxi-server/internal/config/payment"
	"github.com/2comjie/taoxi-server/pkg/asynqx"

	paymentStore "github.com/2comjie/taoxi-server/app/Api/payment/internal/store"
	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx/logdef"
)

func purchaseOrderWithoutLock(logCtx logdef.ILogger, ctx context.Context, channel PaymentChannel, order *paymentent.PaymentOrder, thirdPartyOrder *paymentTypes.ThirdPartyOrder) (paymentTypes.OrderProcessResult, *stderr.Error) {
	logCtx.Infof("开始执行支付成功后的发货流水线")
	if stdErr := stepGrantRewards(logCtx, ctx, order); stdErr != nil {
		logCtx.Errorf("发放奖励失败 err=%v", stdErr)
		return paymentTypes.OrderProcessWaiting, stdErr
	}

	// 扭转订单状态 发货完成
	logCtx.Infof("扭转订单状态为 Purchased")
	err := paymentStore.MarkOrderPurchased(ctx, order.ID, thirdPartyOrder)
	if err != nil {
		logCtx.Errorf("扭转订单状态失败 err=%v", err)
		return paymentTypes.OrderProcessWaiting, stderr.InternalServerError("扭转订单状态失败")
	}

	// 完成第三方订单 比如 Google Consume
	logCtx.Infof("完成第三方订单")
	if stdErr := stepComplete(logCtx, ctx, channel, thirdPartyOrder); stdErr != nil {
		logCtx.Errorf("完成第三方订单失败 err=%v", stdErr)
		return paymentTypes.OrderProcessWaiting, stdErr
	}
	logCtx.Infof("支付成功流水线执行完毕")
	return paymentTypes.OrderProcessPurchased, nil
}

func cancelOrderWithoutLock(logCtx logdef.ILogger, ctx context.Context, order *paymentent.PaymentOrder, thirdPartyOrder *paymentTypes.ThirdPartyOrder) (paymentTypes.OrderProcessResult, *stderr.Error) {
	if err := paymentStore.MarkOrderCancelled(ctx, order.ID, thirdPartyOrder); err != nil {
		logCtx.Errorf("取消支付订单失败 err=%v", err)
		return paymentTypes.OrderProcessWaiting, stderr.InternalServerError("取消支付订单失败")
	}

	logCtx.Infof("支付订单取消成功")
	return paymentTypes.OrderProcessCancelled, nil
}

func revokeOrderWithoutLock(logCtx logdef.ILogger, ctx context.Context, order *paymentent.PaymentOrder, thirdPartyOrder *paymentTypes.ThirdPartyOrder) (paymentTypes.OrderProcessResult, *stderr.Error) {
	logCtx.Infof("开始执行支付退款奖励回收流水线")
	if stdErr := stepRevokeRewards(logCtx, ctx, order); stdErr != nil {
		logCtx.Errorf("提交奖励回收任务失败 err=%v", stdErr)
		return paymentTypes.OrderProcessPurchased, stdErr
	}

	if err := paymentStore.MarkOrderRefunded(ctx, order.ID, thirdPartyOrder); err != nil {
		logCtx.Errorf("更新订单退款状态失败 err=%v", err)
		return paymentTypes.OrderProcessPurchased, stderr.InternalServerError("更新订单退款状态失败")
	}

	logCtx.Infof("支付退款奖励回收流水线执行完毕")
	return paymentTypes.OrderProcessCancelled, nil
}

func stepGrantRewards(logCtx logdef.ILogger, ctx context.Context, order *paymentent.PaymentOrder) *stderr.Error {
	logCtx.Infof("开始发货")

	product := paymentConfig.GetProduct(order.ProductID)
	if product == nil {
		logCtx.Errorf("发货失败 商品配置不存在 product_id=%d", order.ProductID)
		return stderr.InternalServerError("商品配置不存在")
	}

	rewards, err := json.Marshal(product.Rewards)
	if err != nil {
		logCtx.Errorf("发货失败 序列化奖励错误 err=%v", err)
		return stderr.InternalServerError("序列化奖励失败")
	}

	logCtx.Infof("更新订单奖励快照 rewards=%s", string(rewards))
	if err = paymentStore.UpdateOrderRewards(ctx, order.ID, rewards); err != nil {
		logCtx.Errorf("发货失败 更新奖励快照错误 err=%v", err)
		return stderr.InternalServerError("更新奖励快照失败")
	}

	logCtx.Infof("提交发货任务")
	if err = asynqx.Enqueue(ctx, &paymentTypes.GrantTask{
		OrderId: order.ID,
		Uid:     order.UID,
		Rewards: rewards,
	}); err != nil {
		logCtx.Errorf("提交发货任务失败 err=%v", err)
		return stderr.InternalServerError("提交发货任务失败")
	}

	logCtx.Infof("发货任务提交成功")
	return nil
}

func stepRevokeRewards(logCtx logdef.ILogger, ctx context.Context, order *paymentent.PaymentOrder) *stderr.Error {
	logCtx.Infof("提交奖励回收任务 rewards=%s", string(order.Rewards))
	if err := asynqx.Enqueue(ctx, &paymentTypes.RevokeTask{
		OrderId: order.ID,
		Uid:     order.UID,
		Rewards: order.Rewards,
	}); err != nil {
		logCtx.Errorf("提交奖励回收任务失败 err=%v", err)
		return stderr.InternalServerError("提交奖励回收任务失败")
	}

	logCtx.Infof("奖励回收任务提交成功")
	return nil
}

func stepComplete(logCtx logdef.ILogger, ctx context.Context, channel PaymentChannel, thirdPartyOrder *paymentTypes.ThirdPartyOrder) *stderr.Error {
	if err := channel.Complete(ctx, thirdPartyOrder); err != nil {
		logCtx.Errorf("完成第三方订单失败 err=%v", err)
		return stderr.InternalServerError("完成第三方订单失败")
	}
	return nil
}
