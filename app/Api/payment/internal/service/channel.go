package paymentService

import (
	"context"

	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
)

type PaymentChannel interface {
	// Type 返回支付渠道类型
	Type() paymentTypes.PaymentType

	// BuildCreateOrderExtra 构建客户端拉起支付需要的渠道参数
	BuildCreateOrderExtra(ctx context.Context, params *paymentTypes.BuildCreateOrderParams) (*paymentTypes.CreateOrderExtra, error)

	// ParseCallback 验证并解析第三方支付回调
	// 没有实际订单的测试通知可以返回 nil, nil
	ParseCallback(ctx context.Context, req *paymentTypes.CallbackRequest) (*paymentTypes.CallbackEvent, error)

	// QueryOrder 查询并验证第三方真实订单，返回统一订单信息
	QueryOrder(ctx context.Context, credential string) (*paymentTypes.ThirdPartyOrder, error)

	// Complete 在内部发奖成功后完成第三方订单
	// Google执行consume，Apple可以直接返回nil
	Complete(ctx context.Context, order *paymentTypes.ThirdPartyOrder) error

	// FindOrBind 查询或绑定第三方订单
	FindOrBind(ctx context.Context, uid uint64, credential string, info *paymentTypes.ThirdPartyOrder) (*paymentent.PaymentOrder, error)
}

var channels = make(map[paymentTypes.PaymentType]PaymentChannel)

func RegisterChannel(channel PaymentChannel) {
	if channel == nil {
		panic("payment: PaymentChannel不能为空")
	}
	channels[channel.Type()] = channel
}

func GetChannel(paymentType paymentTypes.PaymentType) (PaymentChannel, bool) {
	channel, exists := channels[paymentType]
	return channel, exists
}
