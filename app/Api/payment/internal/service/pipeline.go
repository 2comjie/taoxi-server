package service

import (
	"context"

	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx/logdef"
)

// purchaseOrderWithoutLock 执行支付成功后的发货流水线。
// 发奖能力接入前保持订单原状态，不能把未发奖订单标记为Purchased。
func purchaseOrderWithoutLock(logCtx logdef.ILogger, ctx context.Context, channel PaymentChannel, order *paymentent.PaymentOrder, thirdPartyOrder *paymentTypes.ThirdPartyOrder) (paymentTypes.OrderProcessResult, *stderr.Error) {
	if stdErr := stepGrantRewards(logCtx, ctx, order); stdErr != nil {
		return paymentTypes.OrderProcessWaiting, stdErr
	}

	logCtx.Warnf("支付订单状态更新流水线暂未实现 order_id=%d", order.ID)
	return paymentTypes.OrderProcessWaiting, stderr.InternalServerError("支付发货服务暂未就绪")
}

// cancelOrderWithoutLock 执行未发货订单的取消流水线。
func cancelOrderWithoutLock(
	logCtx logdef.ILogger,
	ctx context.Context,
	order *paymentent.PaymentOrder,
	thirdPartyOrder *paymentTypes.ThirdPartyOrder,
) (paymentTypes.OrderProcessResult, *stderr.Error) {
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

// stepGrantRewards 幂等发放订单奖励。
// 后续接入asyncq时使用order.ID作为任务幂等标识，并使用order.Rewards奖励快照。
func stepGrantRewards(
	logCtx logdef.ILogger,
	ctx context.Context,
	order *paymentent.PaymentOrder,
) *stderr.Error {
	logCtx.Warnf("支付发奖步骤暂未实现 order_id=%d", order.ID)
	return stderr.InternalServerError("支付发奖服务暂未就绪")
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

// stepComplete 完成第三方订单。Google执行Consume，Apple通常直接返回nil。
func stepComplete(
	logCtx logdef.ILogger,
	ctx context.Context,
	channel PaymentChannel,
	thirdPartyOrder *paymentTypes.ThirdPartyOrder,
) *stderr.Error {
	if err := channel.Complete(ctx, thirdPartyOrder); err != nil {
		logCtx.Errorf("完成第三方订单失败 err=%v", err)
		return stderr.InternalServerError("完成第三方订单失败")
	}
	return nil
}
