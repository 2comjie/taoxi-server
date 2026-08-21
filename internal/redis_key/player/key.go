package playerRedisKey

import "fmt"

func Diffs(uid uint64) string {
	return fmt.Sprintf("player:{%d}:diffs", uid)
}
