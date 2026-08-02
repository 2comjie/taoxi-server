package shared

type Reward struct {
	ItemTypeId        int32
	Count             int32
	ExpireDurationSec int64
	ExpireTimeUnix    int64
	Args              string
}
