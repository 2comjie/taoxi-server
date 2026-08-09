package itemRedisKey

import "fmt"

func UserItemVersion(uid uint64) string {
	return fmt.Sprintf("item:bag:ver:{%d}", uid)
}
