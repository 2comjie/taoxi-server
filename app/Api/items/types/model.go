package itemTypes

import "github.com/2comjie/taoxi-server/internal/config/items"

type OperationType int32

const (
	OperationTypeGrant  OperationType = 1
	OperationTypeUse    OperationType = 2
	OperationTypeRevoke OperationType = 3
)

type Item struct {
	BagId        int64            `json:"bag_id"`
	ItemTypeId   items.ItemTypeId `json:"item_id"`
	Quantity     int64            `json:"quantity"`
	ExpireAtUnix int64            `json:"expire_at_unix"`
}
