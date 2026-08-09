package paymentConfig

import (
	"maps"

	"github.com/2comjie/wali/config"
)

const productConfigKey = "payment.product"

type productMap map[int32]*Product

var products config.WatchedValue[productMap]

func Init(center config.Config) error {
	return products.Init(center, productConfigKey)
}

func GetProduct(id int32) *Product {
	return products.Load()[id]
}

func GetAllProducts() map[int32]*Product {
	return maps.Clone(products.Load())
}
