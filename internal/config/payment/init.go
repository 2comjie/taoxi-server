package paymentConfig

import (
	"maps"

	"github.com/2comjie/nova/config"
)

const productConfigKey = "payment.product"
const googleConfigKey = "payment.google"

type productMap map[int32]*Product

var products config.WatchedValue[productMap]
var googleConfig config.WatchedValue[Google]

func Init(center config.Config) error {
	if err := products.Init(center, productConfigKey); err != nil {
		return err
	}
	return googleConfig.Init(center, googleConfigKey)
}

func GetProduct(id int32) *Product {
	return products.Load()[id]
}

func GetAllProducts() map[int32]*Product {
	return maps.Clone(products.Load())
}

func GetGoogle() Google {
	return googleConfig.Load()
}
