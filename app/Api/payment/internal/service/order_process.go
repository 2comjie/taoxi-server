package service

import (
	"context"

	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx/logdef"
)

//内部状态	第三方状态	处理
//Pending	Pending	等待，不修改
//Pending	Purchased	幂等发奖 → Purchased → Complete
//Pending	Cancelled	修改为 Cancelled
//Purchased	Pending	异常告警，不修改，等待重查
//Purchased	Purchased	不重复发奖，重试 Complete
//Purchased	Cancelled	幂等回收 → Cancelled
//Cancelled	Pending	不修改，等待第三方最终状态
//Cancelled	Purchased	仅 timeout 订单允许恢复发奖
//Cancelled	Cancelled	幂等成功

// ProcessOrderWithoutLock 根据内部订单状态和第三方订单状态执行支付流水线。
func ProcessOrderWithoutLock(logCtx logdef.ILogger, ctx context.Context, channel PaymentChannel, order *paymentent.PaymentOrder, thirdPartyOrder *paymentTypes.ThirdPartyOrder) (paymentTypes.OrderProcessResult, *stderr.Error) {
	logCtx = logCtx.
		WithField("order_id", order.ID).
		WithField("third_party_status", thirdPartyOrder.Status).
		WithField("internal_status", order.Status)

	switch order.Status {
	case paymentTypes.OrderStatusPending:
		switch thirdPartyOrder.Status {
		case paymentTypes.ThirdPartyOrderStatusPending:
			logCtx.Infof("内部订单为Pending，第三方订单为Pending，等待回调或者超时处理")
			return paymentTypes.OrderProcessWaiting, nil

		case paymentTypes.ThirdPartyOrderStatusPurchased:
			logCtx.Infof("内部订单为Pending，第三方订单为Purchased，开始执行发货流水线")
			return purchaseOrderWithoutLock(logCtx, ctx, channel, order, thirdPartyOrder)

		case paymentTypes.ThirdPartyOrderStatusCancelled:
			logCtx.Infof("内部订单为Pending，第三方订单为Cancelled，开始撤销内部订单")
			return cancelOrderWithoutLock(logCtx, ctx, order, thirdPartyOrder)

		default:
			logCtx.Errorf("未知第三方订单状态 status=%d", thirdPartyOrder.Status)
			return 0, stderr.InternalServerError("第三方订单状态错误")
		}

	case paymentTypes.OrderStatusPurchased:
		switch thirdPartyOrder.Status {
		case paymentTypes.ThirdPartyOrderStatusPending:
			logCtx.Errorf("内部订单为Purchased，但第三方订单为Pending，需要告警并等待重查")
			return paymentTypes.OrderProcessPurchased, nil

		case paymentTypes.ThirdPartyOrderStatusPurchased:
			logCtx.Infof("内部订单和第三方订单均为Purchased，幂等完成第三方订单")

			if stdErr := stepComplete(logCtx, ctx, channel, thirdPartyOrder); stdErr != nil {
				return 0, stdErr
			}

			return paymentTypes.OrderProcessPurchased, nil

		case paymentTypes.ThirdPartyOrderStatusCancelled:
			logCtx.Infof("内部订单为Purchased，第三方订单为Cancelled，开始回收奖励并撤销内部订单")
			return revokeOrderWithoutLock(logCtx, ctx, order, thirdPartyOrder)

		default:
			logCtx.Errorf("未知第三方订单状态 status=%d", thirdPartyOrder.Status)
			return 0, stderr.InternalServerError("第三方订单状态错误")
		}

	case paymentTypes.OrderStatusCancelled:
		switch thirdPartyOrder.Status {
		case paymentTypes.ThirdPartyOrderStatusPending:
			if order.CancelReason == paymentTypes.CancelReasonTimeout {
				logCtx.Infof("内部订单因超时取消，第三方订单仍为Pending，等待第三方最终状态")
			} else {
				logCtx.Errorf("非超时取消的内部订单，第三方订单为Pending cancel_reason=%s", order.CancelReason)
			}

			return paymentTypes.OrderProcessCancelled, nil

		case paymentTypes.ThirdPartyOrderStatusPurchased:
			if order.CancelReason != paymentTypes.CancelReasonTimeout {
				logCtx.Errorf("非超时取消的内部订单，第三方订单变为Purchased，不允许恢复 cancel_reason=%s", order.CancelReason)
				return 0, stderr.BadRequest("订单已经取消")
			}

			logCtx.Infof("内部订单因超时取消，但第三方订单已经Purchased，重新执行发货流水线")
			return purchaseOrderWithoutLock(logCtx, ctx, channel, order, thirdPartyOrder)

		case paymentTypes.ThirdPartyOrderStatusCancelled:
			logCtx.Infof("内部订单和第三方订单均为Cancelled，幂等处理成功")
			return paymentTypes.OrderProcessCancelled, nil

		default:
			logCtx.Errorf("未知第三方订单状态 status=%d", thirdPartyOrder.Status)
			return 0, stderr.InternalServerError("第三方订单状态错误")
		}

	default:
		logCtx.Errorf("未知内部订单状态 status=%d", order.Status)
		return 0, stderr.InternalServerError("内部订单状态错误")
	}
}
