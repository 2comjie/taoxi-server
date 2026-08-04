package paymentRedisKey

import (
	"fmt"
)

func UserLock(uid uint64) string {
	return fmt.Sprintf("payment:{%d}:user_lock", uid)
}
