package service

import (
	"context"

	paymentStore "github.com/2comjie/taoxi-server/app/Api/payment/internal/store"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	"github.com/2comjie/taoxi-server/pkg/stderr"
)

// 内部状态  第三方状态  可能触发的场景                              处理方式
//
// Pending   Pending     订单刚创建、延迟付款、支付尚未完成          有凭证就保存，保持Pending。
//                                                                  Cron重新查询第三方状态；没有凭证的超时订单才直接关闭。
//
// Pending   Purchased   玩家刚支付成功，或者之前发奖失败            校验订单、商品、UID和凭证，幂等发奖。
//                                                                  成功后改为Purchased，失败继续保持Pending。
//
// Pending   Cancelled   玩家取消、支付失败，或者支付后很快退款       禁止继续发奖，防御性调用幂等RevokeOrder。
//                         但内部订单尚未完成                          回收成功后改为Cancelled，失败保持Pending重试。
//
// Purchased Pending     异常或暂时不一致：内部已经完成发奖，          不立即回收，也不修改内部状态。
//                         但第三方仍显示处理中                        延迟重新查询第三方并告警，等待Purchased或Cancelled。
//
// Purchased Purchased   订单正常完成，或者收到重复通知               直接幂等成功，不重复发奖，回调返回成功。
//
// Purchased Cancelled   退款、拒付、撤销购买，此前已经完成发奖        幂等调用RevokeOrder。
//                                                                  成功后改为Cancelled，失败保持Purchased等待重试。
//
// Cancelled Pending     内部因超时关闭，但第三方支付仍在进行          cancel_reason=timeout时继续等待，不发奖。
//                                                                  其他取消原因出现该组合时重新查询并告警。
//
// Cancelled Purchased   内部超时关闭，但第三方支付成功通知迟到        只有cancel_reason=timeout时允许恢复。
//                                                                  重新校验并幂等发奖，成功后改为Purchased。
//                                                                  refund/manual取消的订单禁止恢复并告警。
//
// Cancelled Cancelled   超时订单最终被平台取消，或者退款回收完成       正常终态，记录第三方取消/退款时间和原因。
//                                                                  重复回调直接返回成功。

type Manager struct {
	store    *paymentStore.Store
	channels map[paymentTypes.PaymentType]PaymentChannel
}

func NewManager(store *paymentStore.Store) *Manager {
	if store == nil {
		panic("payment: Store不能为空")
	}
	return &Manager{
		store:    store,
		channels: make(map[paymentTypes.PaymentType]PaymentChannel),
	}
}

func (m *Manager) CreateOrder(ctx context.Context, uid uint64, req *paymentTypes.CreateOrderReq) (*paymentTypes.CreateOrderRsp, *stderr.Error) {
	// 1. 获取支付渠道

	// 2. 查询并校验商品配置
	// 3. 获取分布式锁
	// 4. 查询可服用的Pending订单
	// 5. 没有则创建订单
	// 6. 构建Apple/Google客户端参数
	// 7. 返回订单

}
