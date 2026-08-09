package sharedItem

type StackMode int32

const (
	StackModeCount StackMode = 0 // 按照数量堆叠 item_type_id + expire_time 相同才会合并
	StackModeTime  StackMode = 1 // 按照时间堆叠 同类型的道具 会叠加时长
)
