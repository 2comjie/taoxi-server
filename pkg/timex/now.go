package timex

import "time"

func NowUnix() int64 {
	return time.Now().Unix()
}
