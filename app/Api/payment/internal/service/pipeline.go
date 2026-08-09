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

// purchaseOrderWithoutLock 执行支付成功后的发货流水线
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

// cancelOrderWithoutLock 执行未发货订单的取消流水线。
func cancelOrderWithoutLock(logCtx logdef.ILogger, ctx context.Context, order *paymentent.PaymentOrder, thirdPartyOrder *paymentTypes.ThirdPartyOrder) (paymentTypes.OrderProcessResult, *stderr.Error) {
	logCtx.Warnf("支付订单取消流水线暂未实现 order_id=%d", order.ID)
	return paymentTypes.OrderProcessWaiting, stderr.InternalServerError("支付订单取消服务暂未就绪")
}

// revokeOrderWithoutLock 执行已发货订单的奖励回收流水线。
// 回收能力接入前保持Purchased，不能提前把订单标记为Cancelled。
func revokeOrderWithoutLock(
	logCtx logdef.ILogger,
	ctx context.Context,
	order *paymentent.PaymentOrder,
	thirdPartyOrder *paymentTypes.ThirdPartyOrder,
) (paymentTypes.OrderProcessResult, *stderr.Error) {
	if stdErr := stepRevokeRewards(logCtx, ctx, order); stdErr != nil {
		return paymentTypes.OrderProcessPurchased, stdErr
	}

	logCtx.Warnf("支付退款状态更新流水线暂未实现 order_id=%d", order.ID)
	return paymentTypes.OrderProcessPurchased, stderr.InternalServerError("支付奖励回收服务暂未就绪")
}

// stepGrantRewards 幂等发放订单奖励
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

// stepRevokeRewards 幂等回收订单奖励。
// 后续接入asyncq时使用order.ID作为任务幂等标识，并使用order.Rewards奖励快照。
func stepRevokeRewards(
	logCtx logdef.ILogger,
	ctx context.Context,
	order *paymentent.PaymentOrder,
) *stderr.Error {
	logCtx.Warnf("支付奖励回收步骤暂未实现 order_id=%d", order.ID)
	return stderr.InternalServerError("支付奖励回收服务暂未就绪")
}

// stepComplete 完成第三方订单。Google执行Consume，Apple通常直接返回nil
func stepComplete(logCtx logdef.ILogger, ctx context.Context, channel PaymentChannel, thirdPartyOrder *paymentTypes.ThirdPartyOrder) *stderr.Error {
	if err := channel.Complete(ctx, thirdPartyOrder); err != nil {
		logCtx.Errorf("完成第三方订单失败 err=%v", err)
		return stderr.InternalServerError("完成第三方订单失败")
	}
	return nil
}
