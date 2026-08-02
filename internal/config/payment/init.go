package paymentConfig

import (
	"fmt"
	"maps"
	"sync/atomic"

	"github.com/2comjie/wali/config"
	"github.com/2comjie/wali/logx"
)

const productConfigKey = "payment.product"

type productMap map[int32]*Product

var products atomic.Pointer[productMap]

func Init(center config.Config) error {
	value := center.Value(productConfigKey)
	if value == nil {
		return fmt.Errorf("payment config: 未找到配置 %s", productConfigKey)
	}
	err := loadProducts(value)
	if err != nil {
		return err
	}

	err = center.Watch(productConfigKey, func(_ string, value config.Value) {
		loadErr := loadProducts(value)
		if loadErr != nil {
			logx.Errorf("payment config: 重载商品配置失败 err=%v", loadErr)
		}
	})
	if err != nil {
		return err
	}
	return nil
}

func loadProducts(value config.Value) error {
	var current productMap
	if err := value.Scan(&current); err != nil {
		return fmt.Errorf("payment config: 解析 %s 失败: %w", productConfigKey, err)
	}
	if current == nil {
		current = make(productMap)
	}

	products.Store(&current)
	return nil
}

func GetProduct(id int32) *Product {
	snapshot := products.Load()
	if snapshot == nil {
		return nil
	}
	return (*snapshot)[id]
}

func GetAllProducts() map[int32]*Product {
	snapshot := products.Load()
	if snapshot == nil {
		return nil
	}
	return maps.Clone(*snapshot)
}
