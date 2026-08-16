package items

import (
	"github.com/2comjie/nova/config"
	"maps"
)

const itemConfigKey = "item.item"

type itemMap map[ItemTypeId]*Item

var items config.WatchedValue[itemMap]

func Init(center config.Config) error {
	return items.Init(center, itemConfigKey)
}

func GetAllItems() map[ItemTypeId]*Item {
	return maps.Clone(items.Load())
}

func GetItem(itemType ItemTypeId) *Item {
	return items.Load()[itemType]
}
