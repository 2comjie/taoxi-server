package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	"github.com/2comjie/taoxi-server/pkg/timex"
)

type PaymentOrder struct {
	ent.Schema
}

func (PaymentOrder) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").
			Immutable().
			Comment("内部订单ID"),

		field.Uint64("uid").
			Immutable().
			Comment("uid"),

		field.Int32("product_id").
			Immutable().
			Comment("内部商品ID"),

		field.String("third_party_product_id").
			Immutable().
			Comment("第三方商品ID"),

		field.Int32("payment_type").
			Immutable().
			Comment("支付渠道"),

		field.Int32("status").
			GoType(paymentTypes.OrderStatus(0)).
			Default(int32(paymentTypes.OrderStatusPending)).
			Comment("内部订单状态"),

		// 下单时候的价格信息快照
		field.Int64("order_amount_unit").
			Immutable().
			Comment("下单金额整数部分"),

		field.Int32("order_amount_nanos").
			Immutable().
			Comment("下单金额小数部分"),

		field.String("order_currency").
			Immutable().
			Comment("下单货币"),

		// 第三方查询得到的实际支付金额
		field.Int64("real_amount_unit").
			Default(0).
			Comment("实际支付金额整数部分"),

		field.Int32("real_amount_nanos").
			Default(0).
			Comment("实际支付金额小数部分"),

		field.String("real_currency").
			Default("").
			Comment("实际支付货币"),

		field.String("third_party_order_id").
			Optional().
			Nillable().
			Comment("第三方订单号"),

		// 原始凭证用于第三方查询和Cron重试
		field.Text("credential").
			Optional().
			Nillable().
			Comment("Apple交易凭证或Google purchaseToken"),

		// 唯一索引使用固定长度Hash，避免给长凭证建立索引
		field.String("credential_hash").
			Optional().
			Nillable().
			Comment("支付凭证SHA-256"),

		// 下单时保存奖励快照，退款时不能再读取可能已经修改的商品配置
		field.JSON("rewards", json.RawMessage{}).
			Optional().
			Comment("订单奖励快照"),

		// 重试次数
		field.Int32("retry_times").
			Default(0).
			Comment("处理重试次数"),

		field.String("cancel_reason").
			Default("").
			Comment("timeout或refund"),

		field.String("refund_reason").
			Default("").
			Comment("第三方退款原因"),

		field.Bool("is_sandbox").
			Default(false).
			Comment("是否沙箱订单"),

		field.Int64("create_at_unix").
			DefaultFunc(timex.NowUnix).
			Immutable().
			Comment("创建时间，Unix秒"),

		field.Int64("update_at_unix").
			DefaultFunc(timex.NowUnix).
			UpdateDefault(timex.NowUnix).
			Comment("更新时间，Unix秒"),

		field.Int64("pay_at_unix").
			Default(0).
			Comment("支付完成时间，Unix秒"),

		field.Int64("refund_at_unix").
			Default(0).
			Comment("第三方退款时间，Unix秒"),

		field.Int64("cancel_at_unix").
			Default(0).
			Comment("内部订单关闭时间，Unix秒"),
	}
}

func (PaymentOrder) Indexes() []ent.Index {
	return []ent.Index{
		// 创建订单时查找并复用Pending订单
		index.Fields(
			"uid",
			"payment_type",
			"product_id",
			"status",
		),

		// Cron扫描超时Pending订单
		index.Fields(
			"status",
			"create_at_unix",
		),

		// 查询玩家订单记录
		index.Fields(
			"uid",
			"create_at_unix",
		),

		// 防止同一第三方订单绑定多个内部订单
		index.Fields(
			"payment_type",
			"third_party_order_id",
		).Unique(),

		// 防止同一支付凭证被重复使用
		index.Fields(
			"payment_type",
			"credential_hash",
		).Unique(),
	}
}

func (PaymentOrder) Edges() []ent.Edge {
	return nil
}
