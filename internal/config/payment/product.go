package paymentConfig

import "github.com/2comjie/taoxi-server/internal/config/shared"

type Price struct {
	AmountUnit  int64 `json:"amount_unit"`
	AmountNanos int32 `json:"amount_nanos"`
}

type Product struct {
	Id                  int32
	ThirdPartyProductId map[int32]string
	PriceMap            map[string]*Price
	Rewards             []*shared.Reward
}
