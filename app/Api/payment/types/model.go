package paymentTypes

import "net/http"

type OrderStatus int32

const (
	OrderStatusPending   OrderStatus = 0
	OrderStatusPurchased OrderStatus = 1
	OrderStatusCancelled OrderStatus = 2
)

const (
	CancelReasonTimeout = "timeout"
	CancelReasonRefund  = "refund"
)

type PaymentType int32

const (
	PaymentTypeApple  PaymentType = 1
	PaymentTypeGoogle PaymentType = 2
)

type ThirdPartyOrderStatus int32

const (
	ThirdPartyOrderStatusPending   ThirdPartyOrderStatus = 0
	ThirdPartyOrderStatusPurchased ThirdPartyOrderStatus = 1
	ThirdPartyOrderStatusCancelled ThirdPartyOrderStatus = 2
)

type BuildCreateOrderParams struct {
	OrderId             uint64
	Uid                 uint64
	ProductId           int32  // 商品ID
	ThirdPartyProductId string // 第三方的商品ID
}
type CreateOrderExtra struct {
	// Apple
	AppAccountToken string `json:"app_account_token,omitempty"`

	// Google
	ObfuscatedAccountID string `json:"obfuscated_account_id,omitempty"`
	ObfuscatedProfileID string `json:"obfuscated_profile_id,omitempty"`
}

type CallbackRequest struct {
	Header http.Header
	Body   []byte
}

type CallbackEvent struct {
	Credential string
}

// 第三方订单信息
type ThirdPartyOrder struct {
	Status ThirdPartyOrderStatus

	// 内部订单定位信息
	InternalOrderId uint64
	Uid             uint64

	// 第三方订单号 (如 Google 的 GPA.xxx)
	OrderId    string
	ProductId  string
	Credential string

	AmountUnit  int64
	AmountNanos int32
	Currency    string

	PayAtUnix    int64
	RefundAtUnix int64
	RefundReason string

	IsSandbox bool
}
