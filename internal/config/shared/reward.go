package shared

import "github.com/2comjie/taoxi-server/internal/config/items"

type Reward struct {
	ItemTypeId        items.ItemTypeId
	Count             int32
	ExpireDurationSec int64
	ExpireTimeUnix    int64
	Args              string
}
