package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	"github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent/paymentorder"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
)

var EntClient *paymentent.Client

func Init(driver *entsql.Driver) {
	if driver == nil {
		panic("payment store: Ent Driver不能为空")
	}
	EntClient = paymentent.NewClient(paymentent.Driver(driver))
}

func Migrate(ctx context.Context) error {
	if err := EntClient.Schema.Create(ctx); err != nil {
		return fmt.Errorf("payment: 创建支付订单表失败: %w", err)
	}
	return nil
}

func FindPendingOrder(ctx context.Context, uid uint64, paymentType paymentTypes.PaymentType, productId int32) (*paymentent.PaymentOrder, bool, error) {
	order, err := EntClient.PaymentOrder.Query().Where(
		paymentorder.UIDEQ(uid),
		paymentorder.PaymentTypeEQ(paymentType),
		paymentorder.ProductIDEQ(productId),
		paymentorder.StatusEQ(paymentTypes.OrderStatusPending),
	).Order(paymentent.Desc(paymentorder.FieldID)).First(ctx)
	if paymentent.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("payment: 查询未完成订单失败: %w", err)
	}
	return order, true, nil
}

func CreateOrder(ctx context.Context, params *paymentTypes.CreateOrderParams) (*paymentent.PaymentOrder, error) {
	builder := EntClient.PaymentOrder.Create().
		SetUID(params.Uid).
		SetProductID(params.ProductId).
		SetThirdPartyProductID(params.ThirdPartyProductId).
		SetPaymentType(params.PaymentType).
		SetStatus(paymentTypes.OrderStatusPending).
		SetOrderAmountUnit(params.AmountUnit).
		SetOrderAmountNanos(params.AmountNanos).
		SetOrderCurrency(params.Currency)
	if len(params.Rewards) > 0 {
		builder.SetRewards(params.Rewards)
	}

	order, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: 创建订单失败: %w", err)
	}
	return order, nil
}

func FindExpiredPendingOrders(ctx context.Context, expiredAtUnix int64, limit int) ([]*paymentent.PaymentOrder, error) {
	orders, err := EntClient.PaymentOrder.Query().Where(
		paymentorder.StatusEQ(paymentTypes.OrderStatusPending),
		paymentorder.CreateAtUnixLTE(expiredAtUnix),
	).Order(paymentent.Asc(paymentorder.FieldID)).Limit(limit).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: 查询超时订单失败: %w", err)
	}
	return orders, nil
}

func GetExpiredPendingOrder(ctx context.Context, orderId uint64, expiredAtUnix int64) (*paymentent.PaymentOrder, error) {
	order, err := EntClient.PaymentOrder.Query().Where(
		paymentorder.IDEQ(orderId),
		paymentorder.StatusEQ(paymentTypes.OrderStatusPending),
		paymentorder.CreateAtUnixLTE(expiredAtUnix),
	).Only(ctx)
	if paymentent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("payment: 查询超时订单失败: %w", err)
	}
	return order, nil
}

func MarkOrderTimeout(ctx context.Context, orderId uint64, expiredAtUnix int64) error {
	_, err := EntClient.PaymentOrder.Update().Where(
		paymentorder.IDEQ(orderId),
		paymentorder.StatusEQ(paymentTypes.OrderStatusPending),
		paymentorder.CreateAtUnixLTE(expiredAtUnix),
	).
		SetStatus(paymentTypes.OrderStatusCancelled).
		SetCancelReason(paymentTypes.CancelReasonTimeout).
		SetCancelAtUnix(time.Now().Unix()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("payment: 关闭超时订单失败: %w", err)
	}
	return nil
}

func MarkOrderPurchased(ctx context.Context, orderId uint64, thirdPartyOrder *paymentTypes.ThirdPartyOrder) error {
	builder := EntClient.PaymentOrder.Update().
		Where(
			paymentorder.IDEQ(orderId),
			paymentorder.Or(
				paymentorder.StatusEQ(paymentTypes.OrderStatusPending),

				// 超时取消的订单 可以允许恢复订单
				paymentorder.And(
					paymentorder.StatusEQ(paymentTypes.OrderStatusCancelled),
					paymentorder.CancelReasonEQ(paymentTypes.CancelReasonTimeout),
				),
			),
		).
		SetStatus(paymentTypes.OrderStatusPurchased).
		SetRealAmountNanos(thirdPartyOrder.AmountNanos).
		SetRealAmountUnit(thirdPartyOrder.AmountUnit).
		SetRealCurrency(thirdPartyOrder.Currency).
		SetIsSandbox(thirdPartyOrder.IsSandbox).
		SetCancelReason("").
		SetCancelAtUnix(0)

	if thirdPartyOrder.PayAtUnix > 0 {
		builder.SetPayAtUnix(thirdPartyOrder.PayAtUnix)
	}
	if thirdPartyOrder.Credential != "" {
		builder.SetCredential(thirdPartyOrder.Credential)
	}
	affected, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("payment: 订单支付成功失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("payment: 订单支付成功失败: 订单不存在")
	}
	return nil
}

func MarkOrderRefunded(ctx context.Context, orderId uint64, thirdPartyOrder *paymentTypes.ThirdPartyOrder) error {
	refundAtUnix := thirdPartyOrder.RefundAtUnix
	if refundAtUnix <= 0 {
		refundAtUnix = time.Now().Unix()
	}
	refundReason := thirdPartyOrder.RefundReason
	if refundReason == "" {
		refundReason = paymentTypes.CancelReasonRefund
	}

	affected, err := EntClient.PaymentOrder.Update().
		Where(
			paymentorder.IDEQ(orderId),
			paymentorder.StatusEQ(paymentTypes.OrderStatusPurchased),
		).
		SetStatus(paymentTypes.OrderStatusCancelled).
		SetCancelReason(paymentTypes.CancelReasonRefund).
		SetRefundReason(refundReason).
		SetRefundAtUnix(refundAtUnix).
		SetCancelAtUnix(refundAtUnix).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("payment: 更新订单退款状态失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("payment: 更新订单退款状态失败: 订单不存在或状态已改变")
	}
	return nil
}

func MarkOrderCancelled(ctx context.Context, orderId uint64, thirdPartyOrder *paymentTypes.ThirdPartyOrder) error {
	cancelAtUnix := time.Now().Unix()
	if thirdPartyOrder.RefundAtUnix > 0 {
		cancelAtUnix = thirdPartyOrder.RefundAtUnix
	}

	builder := EntClient.PaymentOrder.Update().
		Where(
			paymentorder.IDEQ(orderId),
			paymentorder.StatusEQ(paymentTypes.OrderStatusPending),
		).
		SetStatus(paymentTypes.OrderStatusCancelled).
		SetCancelReason(paymentTypes.CancelReasonRefund).
		SetCancelAtUnix(cancelAtUnix)

	if thirdPartyOrder.RefundAtUnix > 0 {
		builder.SetRefundAtUnix(thirdPartyOrder.RefundAtUnix)
	}
	if thirdPartyOrder.RefundReason != "" {
		builder.SetRefundReason(thirdPartyOrder.RefundReason)
	}

	affected, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("payment: 取消订单失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("payment: 取消订单失败: 订单不存在或状态已改变")
	}

	return nil
}

func AddRetryTimes(ctx context.Context, orderId uint64) error {
	_, err := EntClient.PaymentOrder.Update().Where(
		paymentorder.IDEQ(orderId),
	).AddRetryTimes(1).Save(ctx)
	if err != nil {
		return fmt.Errorf("payment: 增加订单重试次数失败: %w", err)
	}
	return nil
}

func UpdateOrderRewards(ctx context.Context, orderId uint64, rewards json.RawMessage) error {
	_, err := EntClient.PaymentOrder.UpdateOneID(orderId).SetRewards(rewards).Save(ctx)
	if err != nil {
		return err
	}
	return nil
}
