package store

import (
	"context"
	"fmt"

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
