package items

type StackMode int32

const (
	StackModeCount StackMode = 0 // 按照数量堆叠 item_type_id + expire_time 相同才会合并
	StackModeTime  StackMode = 1 // 按照时间堆叠 同类型的道具 会叠加时长
)

type ItemSystemId int32
type ItemTypeId int32

type ItemLevel int32

const (
	Coin ItemSystemId = 1
	Seed ItemSystemId = 2
)

type Item struct {
	ItemTypeId   ItemTypeId   `json:"item_type_id"`
	ItemSystemId ItemSystemId `json:"item_system_id"`
	Level        ItemLevel    `json:"level"`
	Name         string       `json:"name"`
	Desc         string       `json:"desc"`
	StackMode    StackMode    `json:"stack_mode"`
	StackCnt     int32        `json:"stack_cnt"`
	ExtraArgs    string       `json:"extra_args"`
}
