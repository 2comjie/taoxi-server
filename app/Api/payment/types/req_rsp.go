package paymentTypes

type CreateOrderReq struct {
	ProductId   int32       `json:"product_id" binding:"required"`
	PaymentType PaymentType `json:"payment_type" binding:"required"`
}
type CreateOrderRsp struct {
	OrderId             uint64            `json:"order_id"`
	ProductId           int32             `json:"product_id"`
	ThirdPartyProductId string            `json:"third_party_product_id"`
	Extra               *CreateOrderExtra `json:"extra,omitempty"`
}

type UploadReceiptReq struct {
	PaymentType PaymentType `json:"payment_type" binding:"required"`
	Credential  string      `json:"credential" binding:"required"`
}

type UploadReceiptRsp struct{}
